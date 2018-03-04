package binlog

import (
	"sync"
	"sync/atomic"
	"library/services"
	"github.com/siddontang/go-mysql/mysql"
	log "github.com/sirupsen/logrus"
	"library/app"
	"time"
)

func NewBinlog(ctx *app.Context) *Binlog {
	binlog := &Binlog{
		Config   : ctx.MysqlConfig,
		wg       : new(sync.WaitGroup),
		lock     : new(sync.Mutex),
		ctx      : ctx,
		services : make(map[string]services.Service),
		ServiceIp        : ctx.TcpConfig.ServiceIp,
		ServicePort      : ctx.TcpConfig.Port,
		startServiceChan : make(chan struct{}, 100),
		stopServiceChan  : make(chan bool, 100),
		status           : binlogStatusIsNormal | binlogStatusIsStop | cacheHandlerClosed | consulIsFollower | disableConsul,
	}
	binlog.consulInit()
	binlog.handlerInit()
	binlog.lookService()
	go binlog.reloadService()
	go binlog.showMembersService()
	return binlog
}

func (h *Binlog) showMembersService() {
	for {
		select {
		case _, ok := <- h.ctx.ShowMembersChan:
			if !ok {
				return
			}
			if len(h.ctx.ShowMembersRes) < cap(h.ctx.ShowMembersRes) {
				members := h.ShowMembers()
				h.ctx.ShowMembersRes <- members
			}
		case <-h.ctx.Ctx.Done():
			return
		}
	}
}

func (h *Binlog) reloadService() {
	for {
		select {
		case service, ok := <-h.ctx.ReloadDone():
			if !ok {
				return
			}
			h.Reload(service)
			case <-h.ctx.Ctx.Done():
				return
		}
	}
}

func (h *Binlog) Close() {
	if h.status & binlogStatusIsExit > 0 {
		return
	}
	if h.status & binlogStatusIsNormal > 0 {
		h.status ^= binlogStatusIsNormal
		h.status |= binlogStatusIsExit
	}
	log.Warn("binlog service exit")
	h.StopService(true)
	for name, service := range h.services {
		log.Debugf("%s service exit", name)
		service.Close()
	}
	h.closeConsul()
	h.agent.ServiceDeregister(h.sessionId)
	h.wg.Wait()
}

func (h *Binlog) lookService() {
	h.wg.Add(2)
	go func() {
		defer h.wg.Done()
		for {
			select {
			case _, ok := <- h.startServiceChan:
				if !ok {
					return
				}
				for {
					if h.status & binlogStatusIsRunning > 0 {
						break
					}
					log.Debug("binlog service start")
					if h.status & binlogStatusIsStop > 0 {
						h.status ^= binlogStatusIsStop
						h.status |= binlogStatusIsRunning
					}
					go func() {
						for {
							if h.lastBinFile == "" {
								log.Warn("binlog lastBinFile is empty, wait for init")
								time.Sleep(time.Second)
								continue
							}
							break
						}
						startPos := mysql.Position{
							Name: h.lastBinFile,
							Pos:  h.lastPos,
						}
						for {
							if h.handler == nil {
								log.Warn("binlog handler is nil, wait for init")
								time.Sleep(time.Second)
								continue
							}
							break
						}
						err := h.handler.RunFrom(startPos)
						if err != nil {
							log.Warnf("binlog service exit with error: %+v", err)
							return
						}
					}()
					break
				}
			case <- h.ctx.Ctx.Done():
				return
			}
		}
	}()
	go func(){
		defer h.wg.Done()
		for {
			select {
			case exit, ok:= <- h.stopServiceChan:
				if !ok {
					return
				}
				if h.status & binlogStatusIsRunning > 0 && !exit {
					log.Debug("binlog service stop")
					h.handler.Close()
					//reset handler
					h.setHandler()
				}
				if exit {
					r := packPos(h.lastBinFile, int64(h.lastPos), atomic.LoadInt64(&h.EventIndex))
					h.SaveBinlogPositionCache(r)
					if h.status & cacheHandlerOpened > 0 {
						h.cacheHandler.Close()
						h.status ^= cacheHandlerOpened
						h.status |= cacheHandlerClosed
					}
				}
				if h.status & binlogStatusIsRunning > 0 {
					h.status ^= binlogStatusIsRunning
					h.status |= binlogStatusIsStop
				}
			case <- h.ctx.Ctx.Done():
				return
			}
		}
	}()
}

func (h *Binlog) StopService(exit bool) {
	h.stopServiceChan <- exit
	if !exit {
		h.agentStart()
	}
}

func (h *Binlog) StartService() {
	h.startServiceChan <- struct{}{}
	for _, s := range h.services {
		s.AgentStop()
	}
}

func (h *Binlog) Start() {
	log.Debugf("===========binlog service start===========")
	for _, service := range h.services {
		service.Start()
	}
	if h.status & disableConsul > 0 {
		log.Debugf("is not enable consul")
		h.StartService()
		return
	}
	go func() {
		for {
			if h.status & binlogStatusIsExit > 0 {
				return
			}
			lock, err := h.Lock()
			if err != nil {
				time.Sleep(time.Second * 3)
				continue
			}
			if lock {
				h.StartService()
			} else {
				h.StopService(false)
			}
			time.Sleep(time.Second * 3)
		}
	}()
}

// start tcp service agent
// service stop will start a tcp service agent
func (h *Binlog) agentStart() {
	serviceIp, port := h.GetLeader()
	currentIp, currentPort := h.GetCurrent()
	if currentIp == serviceIp && currentPort == port {
		log.Debugf("can not start agent with current node %s:%d", currentIp, currentPort)
		return
	}
	if serviceIp == "" || port == 0 {
		log.Warnf("leader ip and port is empty, wait for init, %s:%d", serviceIp, port)
		return
	}
	//log.Debugf("leader ip and port: %s:%d", serviceIp, port)
	if serviceIp == "" || port == 0 {
		return
	}
	for _, s := range h.services {
		s.AgentStart(serviceIp, port)
	}
}

// service reload
// ./wing-binlog-go -service-reload all
// ./wing-binlog-go -service-reload tcp
// ./wing-binlog-go -service-reload http
func (h *Binlog) Reload(service string) {
	if service == serviceNameAll {
		for _, s := range h.services {
			s.Reload()
		}
	} else {
		h.services[service].Reload()
	}
}


