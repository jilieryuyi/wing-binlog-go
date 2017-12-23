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
	"github.com/siddontang/go-mysql/server"
	"sync/atomic"
)

func NewBinlog(ctx *context.Context) *Binlog {
	var b [defaultBufSize]byte
	config, _   := GetMysqlConfig()
	cfg, err    := canal.NewConfigWithFile(file.CurrentPath + "/config/canal.toml")
	if err != nil {
		log.Panicf("binlog create canal config error：%+v", err)
	}
	handler, err := canal.NewCanal(cfg)
	if err != nil {
		log.Panicf("binlog create canal error：%+v", err)
	}

	binlog := &Binlog {
		Config   : config,
		wg       : new(sync.WaitGroup),
		lock     : new(sync.Mutex),
		ctx      : ctx,
		handler  : handler,
		isLeader : true,
		members  : make(map[string]bool),
	}
	cluster := NewCluster(ctx, binlog)
	cluster.Start()
	binlog.BinlogHandler = &binlogHandler{
		services      : make(map[string]services.Service),
		servicesCount : 0,
		lock          : new(sync.Mutex),
		wg            : new(sync.WaitGroup),
		Cluster       : cluster,
		ctx           : ctx,
	}
	f, p, index := binlog.BinlogHandler.getBinlogPositionCache()

	//wait fro SaveBinlogPostionCache complete
	//binlog.BinlogHandler.wg.Add(1)
	binlog.BinlogHandler.Event_index = index
	binlog.BinlogHandler.buf = b[:0]
	binlog.BinlogHandler.isClosed = false

	binlog.handler.SetEventHandler(binlog.BinlogHandler)
	binlog.isClosed = true

	current_pos, err:= binlog.handler.GetMasterPos()
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

func (h *Binlog) setMember(dns string, isLeader bool) {
	log.Debugf("set member: %s, %t", dns, isLeader)
	h.lock.Lock()
	defer h.lock.Unlock()
	h.members[dns] = isLeader
}

func (h *Binlog) getLeader() string  {
	h.lock.Lock()
	defer h.lock.Unlock()
	for dns, isLeader:= range h.members  {
		if isLeader {
			return dns
		}
	}
	return ""
}

func (h *Binlog) Close() {
	log.Warn("binlog service exit")
	if h.isClosed  {
		return
	}
	h.BinlogHandler.lock.Lock()
	h.isClosed = true
	if !h.BinlogHandler.isClosed {
		h.handler.Close()
	}
	data := fmt.Sprintf("%s:%d:%d", h.BinlogHandler.lastBinFile, h.BinlogHandler.lastPos, atomic.LoadInt64(&h.BinlogHandler.Event_index))
	h.BinlogHandler.SaveBinlogPostionCache(data)
	h.BinlogHandler.isClosed = true
	h.BinlogHandler.cacheHandler.Close()
	h.BinlogHandler.Cluster.Close()
	h.BinlogHandler.lock.Unlock()
	for _, service := range h.BinlogHandler.services {
		service.Close()
	}
}

func (h *Binlog) StopService() {
	log.Debug("binlog service stop")
	h.BinlogHandler.lock.Lock()
	defer h.BinlogHandler.lock.Unlock()
	h.BinlogHandler.isClosed = true
	h.handler.Close()
}

func (h *Binlog) StartService() {
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
				log.Warn("binlog service exit: %+v", err)
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
