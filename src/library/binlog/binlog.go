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
		isClosed    : true,
	}

	//init consul
	binlog.consulInit()
	binlog.handlerInit()
	time.Sleep(time.Millisecond * 100)
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
	h.isClosed = false
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
}

func (h *Binlog) Start() {
	log.Debugf("===========binlog service start===========")
	for _, service := range h.services {
		service.Start()
	}
	if h.Lock() {
		log.Debugf("current node will run as leader")
		h.StartService()
	} else {
		go func() {
			var serviceIp= ""
			var port= 0
			for {
				serviceIp, port = h.GetLeader()
				if serviceIp == "" || port == 0 {
					log.Warnf("leader ip and port is empty, wait for init")
					time.Sleep(time.Second)
					continue
				}
				log.Debugf("leader ip and port: %s:%d", serviceIp, port)
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


