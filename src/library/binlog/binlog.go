package binlog

import (
	"github.com/siddontang/go-mysql/canal"
	"github.com/siddontang/go-mysql/mysql"
	"os"
	log "github.com/sirupsen/logrus"
	"library/file"
	"library/services"
	"context"
	"sync"
	"fmt"
	"sync/atomic"
)

func NewBinlog(ctx *context.Context) *Binlog {
	config, _   := GetMysqlConfig()
	var (
		b [defaultBufSize]byte
	 	err error
	)
	binlog := &Binlog {
		Config   : config,
		wg       : new(sync.WaitGroup),
		lock     : new(sync.Mutex),
		ctx      : ctx,
		isLeader : true,
		members  : make(map[string]*member),
	}
	cluster := NewCluster(ctx, binlog)
	cluster.Start()
	binlog.setMember(fmt.Sprintf("%s:%d", cluster.ServiceIp, cluster.port), true)

	binlog.BinlogHandler = &binlogHandler{
		services      : make(map[string]services.Service),
		servicesCount : 0,
		lock          : new(sync.Mutex),
		Cluster       : cluster,
		ctx           : ctx,
	}
	f, p, index := binlog.BinlogHandler.getBinlogPositionCache()

	binlog.BinlogHandler.Event_index = index
	binlog.BinlogHandler.buf = b[:0]
	binlog.BinlogHandler.isClosed = false

	binlog.isClosed = true

	binlog.handler = newCanal()
	binlog.handler.SetEventHandler(binlog.BinlogHandler)


	current_pos, err := binlog.handler.GetMasterPos()
	if f != "" {
		binlog.Config.BinFile = f
	} else {
		if err != nil {
			log.Panicf("binlog get cache error：%+v", err)
		} else {
			binlog.Config.BinFile = current_pos.Name
		}
	}
	if p > 0 {
		binlog.Config.BinPos = p
	} else {
		if err != nil {
			log.Panicf("binlog get cache error：%+v", err)
		} else {
			binlog.Config.BinPos = int64(current_pos.Pos)
		}
	}


	binlog.BinlogHandler.lastBinFile = binlog.Config.BinFile
	binlog.BinlogHandler.lastPos = uint32(binlog.Config.BinPos)

	mysqlBinlogCacheFile := file.CurrentPath +"/cache/mysql_binlog_position.pos"
	dir := file.WPath{mysqlBinlogCacheFile}
	dir  = file.WPath{dir.GetParent()}
	dir.Mkdir()
	flag := os.O_WRONLY | os.O_CREATE | os.O_SYNC | os.O_TRUNC
	binlog.BinlogHandler.cacheHandler, err = os.OpenFile(mysqlBinlogCacheFile, flag , 0755)
	if err != nil {
		log.Panicf("binlog open cache file error：%s, %+v", mysqlBinlogCacheFile, err)
	}
	return binlog
}


func newCanal() (*canal.Canal) {
	cfg, err    := canal.NewConfigWithFile(file.CurrentPath + "/config/canal.toml")
	if err != nil {
		log.Panicf("binlog create canal config error：%+v", err)
	}
	handler, err := canal.NewCanal(cfg)
	if err != nil {
		log.Panicf("binlog create canal error：%+v", err)
	}
	return handler
}


// set member
func (h *Binlog) setMember(dns string, isLeader bool) {
	log.Debugf("set member: %s, %t", dns, isLeader)
	h.lock.Lock()
	defer h.lock.Unlock()
	index := len(h.members) + 1
	h.members[dns] = &member{
		isLeader:isLeader,
		index:index,
	}
}

// get leader dns
func (h *Binlog) getLeader() string  {
	h.lock.Lock()
	defer h.lock.Unlock()
	for dns, member:= range h.members  {
		if member.isLeader {
			return dns
		}
	}
	return ""
}

// set current isLeader
func (h *Binlog) leader(isLeader bool) {
	log.Debugf("binlog set leader %t", isLeader)
	h.lock.Lock()
	defer h.lock.Unlock()
	h.isLeader = isLeader
	dns := fmt.Sprintf("%s:%d", h.BinlogHandler.Cluster.ServiceIp, h.BinlogHandler.Cluster.port)
	v, ok := h.members[dns]
	if ok {
		log.Debugf("%s is leader now", dns)
		v.isLeader = isLeader
	}
}

func (h *Binlog) isNextLeader() bool {
	currentDns := fmt.Sprintf("%s:%d", h.BinlogHandler.Cluster.ServiceIp, h.BinlogHandler.Cluster.port)
	currentIndex := 0
	leaderIndex  := 0
	for dns, member:= range h.members  {
		if dns == currentDns {
			currentIndex = member.index
		}
		if member.isLeader {
			leaderIndex = member.index
		}
	}
	return leaderIndex+1 == currentIndex
}

func (h *Binlog) ShowMembers() string {
	h.lock.Lock()
	defer h.lock.Unlock()
	res := fmt.Sprintf("%-25s%s", "service", "isLeader") + "\r\n"
	unit := "node"
	l := len(h.members)
	if l > 1 {
		unit = "nodes"
	}
	res += fmt.Sprintf("%d %s----------------------------\r\n", l, unit)
	for dns, member:= range h.members  {
		isL := "no"
		if member.isLeader {
			isL = "yes"
		}
		res += fmt.Sprintf("%-25s%s", dns, isL) + "\r\n"
	}
	return res
}

func (h *Binlog) Close() {
	log.Warn("binlog service exit")
	if h.isClosed  {
		return
	}
	h.lock.Lock()
	h.isClosed = true
	h.lock.Unlock()
	h.StopService()
	h.BinlogHandler.cacheHandler.Close()
	h.BinlogHandler.Cluster.Close()
	for name, service := range h.BinlogHandler.services {
		log.Debugf("%s service exit", name)
		service.Close()
		//log.Debugf("%s service exit end", name)
	}
}

func (h *Binlog) StopService() {
	log.Debug("binlog service stop")
	h.BinlogHandler.lock.Lock()
	defer h.BinlogHandler.lock.Unlock()
	if !h.BinlogHandler.isClosed {
		h.handler.Close()
	} else {
		data := fmt.Sprintf("%s:%d:%d", h.BinlogHandler.lastBinFile, h.BinlogHandler.lastPos, atomic.LoadInt64(&h.BinlogHandler.Event_index))
		h.BinlogHandler.SaveBinlogPostionCache(data)
	}
	h.BinlogHandler.isClosed = true
	h.isClosed = true
	//reset cacnal handler
	h.handler = newCanal()
	h.handler.SetEventHandler(h.BinlogHandler)
	//debug.PrintStack()
}

func (h *Binlog) StartService() {
	log.Debug("binlog start service")
	h.lock.Lock()
	defer h.lock.Unlock()
	if !h.isClosed {
		log.Debug("binlog service is not in close status")
		return
	}
	go func() {
		startPos := mysql.Position{
			Name: h.Config.BinFile,
			Pos:  uint32(h.Config.BinPos),
		}
		h.isClosed = false
		err := h.handler.RunFrom(startPos)
		if err != nil {
			if !h.isClosed && !h.BinlogHandler.isClosed {
				// 非关闭情况下退出
				log.Errorf("binlog service exit: %+v", err)
			} else {
				log.Warnf("binlog service exit: %+v", err)
			}
			return
		}
	}()
}

func (h *Binlog) Start() {
	for _, service := range h.BinlogHandler.services {
		service.Start()
	}
	h.StartService()
}

func (h *Binlog) Reload(service string) {
	var (
		tcp       = "tcp"
		websocket = "websocket"
		http      = "http"
		kafka     = "kafka"
		all       = "all"
	)
	switch service {
	case tcp:
		log.Debugf("tcp service reload")
		h.BinlogHandler.services["tcp"].Reload()
	case websocket:
		log.Debugf("websocket service reload")
		h.BinlogHandler.services["websocket"].Reload()
	case http:
		log.Debugf("http service reload")
		h.BinlogHandler.services["http"].Reload()
	case kafka:
		log.Debugf("kafka service reload")
	case all:
		log.Debugf("all service reload")
		h.BinlogHandler.services["tcp"].Reload()
		h.BinlogHandler.services["websocket"].Reload()
		h.BinlogHandler.services["http"].Reload()
	}
}
