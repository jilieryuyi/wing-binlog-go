package binlog

import (
	"sync"
	"sync/atomic"
	"library/services"
	"github.com/siddontang/go-mysql/mysql"
	log "github.com/jilieryuyi/logrus"
	"library/app"
	"time"
)


func NewBinlog(ctx *app.Context) *Binlog {
	config, _ := GetMysqlConfig()
	binlog := &Binlog{
		Config   : config,
		wg       : new(sync.WaitGroup),
		lock     : new(sync.Mutex),
		ctx      : ctx,
		//isLeader : true,
		members  : make(map[string]*member),
		services : make(map[string]services.Service),

		//tcp service ip and port
		ServiceIp   : ctx.ServiceIp,
		ServicePort : ctx.ServicePort,
		isClosed    : true,
	}

	//init consul
	binlog.consulInit()
	binlog.handlerInit()
	return binlog
}

func (h *Binlog) Close() {
	log.Warn("binlog service exit")
	if h.isClosed {
		return
	}
	h.isClosed = true
	h.StopService(true)
	h.cacheHandler.Close()
	for name, service := range h.services {
		log.Debugf("%s service exit", name)
		service.Close()
	}
	h.closeConsul()
}

func (h *Binlog) StopService(exit bool) {
	log.Debug("binlog service stop")
	h.lock.Lock()
	defer h.lock.Unlock()
	if !h.isClosed {
		h.handler.Close()
	} else {
		h.SaveBinlogPostionCache(h.lastBinFile,
			int64(h.lastPos),
				atomic.LoadInt64(&h.EventIndex))
	}
	h.isClosed = true
	if !exit {
		//reset handler
		h.setHandler()
	}
}

func (h *Binlog) StartService() {
	log.Debug("binlog service start")
	h.lock.Lock()
	defer h.lock.Unlock()
	if !h.isClosed {
		log.Debug("binlog service is not in close status")
		return
	}
	go func() {
		startPos := mysql.Position{
			Name: h.lastBinFile,
			Pos:  h.lastPos,
		}
		h.isClosed = false
		log.Debugf("handler===%+v", h.handler)
		for {
			if h.handler == nil {
				time.Sleep(time.Second)
				continue
			}
			break
		}
		err := h.handler.RunFrom(startPos)
		if err != nil {
			log.Warnf("binlog service exit: %+v", err)
			return
		}
	}()
}

func (h *Binlog) Start() {
	log.Debugf("===========binlog service start===========")
	for _, service := range h.services {
		service.Start()
	}
	if h.Lock() {
		log.Debugf("current run as leader")
		h.StartService()
	} else {
		go func() {
			var serviceIp= ""
			var port= 0
			for {
				serviceIp, port = h.GetLeader()
				if serviceIp == "" || port == 0 {
					log.Warnf("wait for get leader ip and port")
					time.Sleep(time.Second)
					continue
				}
				log.Debugf("==%s %d==", serviceIp, port)
				break
			}
			for _, s := range h.services {
				s.AgentStart(serviceIp, port)
			}
		}()
	}
}

func (h *Binlog) Reload(service string) {
	if service == "all" {
		for _, s := range h.services {
			s.Reload()
		}
	} else {
		h.services[service].Reload()
	}
}

func (h *Binlog) onNewLeader() {
	log.Debugf("current run as leader, start running")
	h.StartService()
	for _, s := range h.services {
		s.AgentStop()
	}
}


