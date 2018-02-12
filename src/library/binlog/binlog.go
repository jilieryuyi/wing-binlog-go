package binlog

import (
	"sync"
	"sync/atomic"
	"library/services"
	"github.com/siddontang/go-mysql/mysql"
	log "github.com/sirupsen/logrus"
	"library/app"
	"time"
	"math/rand"
)

func NewBinlog(ctx *app.Context) *Binlog {
	config, _ := GetMysqlConfig()
	tcpConfig, _:= services.GetTcpConfig()
	binlog := &Binlog{
		Config   : config,
		wg       : new(sync.WaitGroup),
		lock     : new(sync.Mutex),
		ctx      : ctx,
		services : make(map[string]services.Service),
		//tcp service ip and port
		ServiceIp   : tcpConfig.ServiceIp,
		ServicePort : tcpConfig.Port,
		startServiceChan:make(chan struct{}, 100),
		stopServiceChan:make(chan bool, 100),
		status : binlogStatusIsNormal | binlogStatusIsStop,
	}
	//init consul
	binlog.consulInit()
	binlog.handlerInit()
	binlog.lookService()
	return binlog
}

func (h *Binlog) Close() {
	if h.status & binlogStatusIsExit > 0 {
		//log.Debugf("binlog service is not running")
		return
	}
	if h.status & binlogStatusIsNormal >0 {
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
	close(h.stopServiceChan)
	close(h.startServiceChan)
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
						//log.Debug("binlog service is still running")
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
					h.cacheHandler.Close()
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
	if !h.enable {
		log.Debugf("is not enable consul")
		h.StartService()
		return
	}
	go func() {
		for {
			if h.Lock() {
				//log.Debugf("lock success")
				h.StartService()
			} else {
				//log.Debugf("lock failure")
				h.StopService(false)
			}
			time.Sleep(time.Second * 3)
		}
	}()
}

// start tcp service agent
// service stop will start a tcp service agent
func (h *Binlog) agentStart() {
	var serviceIp = ""
	var port= 0
	go func() {
		st := time.Now().Unix()
		// get leader service ip and port
		// if empty, wait for init
		// max wait time is 60 seconds
		for {
			if (time.Now().Unix() - st) > 60 {
				break
			}
			serviceIp, port = h.GetLeader()
			if serviceIp == "" || port == 0 {
				log.Warnf("leader ip and port is empty, wait for init, %s:%d", serviceIp, port)
				time.Sleep(time.Second * time.Duration(rand.Int31n(6)))
				continue
			}
			//log.Debugf("leader ip and port: %s:%d", serviceIp, port)
			break
		}
		if serviceIp == "" || port == 0 {
			return
		}
		for _, s := range h.services {
			s.AgentStart(serviceIp, port)
		}
	}()
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


