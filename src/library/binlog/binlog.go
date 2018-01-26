package binlog

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"library/path"
	"library/services"
	"github.com/siddontang/go-mysql/canal"
	"github.com/siddontang/go-mysql/mysql"
	log "github.com/sirupsen/logrus"
	"library/cluster"
)

func NewBinlog(ctx *context.Context) *Binlog {
	config, _ := GetMysqlConfig()
	var (
		b   [defaultBufSize]byte
		err error
	)
	binlog := &Binlog{
		Config:   config,
		wg:       new(sync.WaitGroup),
		lock:     new(sync.Mutex),
		ctx:      ctx,
		isLeader: true,
		members:  make(map[string]*member),
		Drive:    nil,
	}
	binlog.BinlogHandler = &binlogHandler{
		services:      make(map[string]services.Service),
		servicesCount: 0,
		lock:          new(sync.Mutex),
		ctx:           ctx,
		binlog:        binlog,
	}
	mysqlBinlogCacheFile := path.CurrentPath + "/cache/mysql_binlog_position.pos"
	//dir := file.WPath{mysqlBinlogCacheFile}
	//dir = file.WPath{dir.GetParent()}
	//dir.Mkdir()

	path.Mkdir(path.GetParent(mysqlBinlogCacheFile))

	flag := os.O_RDWR | os.O_CREATE | os.O_SYNC // | os.O_TRUNC
	binlog.BinlogHandler.cacheHandler, err = os.OpenFile(mysqlBinlogCacheFile, flag, 0755)
	if err != nil {
		log.Panicf("binlog open cache file error：%s, %+v", mysqlBinlogCacheFile, err)
	}
	f, p, index := binlog.BinlogHandler.getBinlogPositionCache()
	binlog.BinlogHandler.EventIndex = index
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
		binlog.Config.BinPos = uint32(p)
	} else {
		if err != nil {
			log.Panicf("binlog get cache error：%+v", err)
		} else {
			binlog.Config.BinPos = current_pos.Pos
		}
	}
	binlog.BinlogHandler.lastBinFile = binlog.Config.BinFile
	binlog.BinlogHandler.lastPos = uint32(binlog.Config.BinPos)
	return binlog
}

func newCanal() *canal.Canal {
	cfg, err := canal.NewConfigWithFile(path.CurrentPath + "/config/canal.toml")
	if err != nil {
		log.Panicf("binlog create canal config error：%+v", err)
	}
	handler, err := canal.NewCanal(cfg)
	if err != nil {
		log.Panicf("binlog create canal error：%+v", err)
	}
	return handler
}

func (h *Binlog) Close() {
	log.Warn("binlog service exit")
	if h.isClosed {
		return
	}
	h.lock.Lock()
	h.isClosed = true
	h.lock.Unlock()
	h.StopService(true)
	h.BinlogHandler.cacheHandler.Close()
	for name, service := range h.BinlogHandler.services {
		log.Debugf("%s service exit", name)
		service.Close()
	}
}

func (h *Binlog) StopService(exit bool) {
	log.Debug("binlog service stop")
	h.BinlogHandler.lock.Lock()
	defer h.BinlogHandler.lock.Unlock()
	if !h.BinlogHandler.isClosed {
		h.handler.Close()
	} else {
		h.BinlogHandler.SaveBinlogPostionCache(h.BinlogHandler.lastBinFile,
			int64(h.BinlogHandler.lastPos),
				atomic.LoadInt64(&h.BinlogHandler.EventIndex))
	}
	h.BinlogHandler.isClosed = true
	h.isClosed = true
	if !exit {
		h.handler = newCanal()
		h.handler.SetEventHandler(h.BinlogHandler)
	}
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
			Name: h.BinlogHandler.lastBinFile,
			Pos:  h.BinlogHandler.lastPos,
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

func (h *Binlog) RegisterDrive(drive cluster.Cluster) {
	h.Drive = drive
}

func (h *Binlog) Start() {
	for _, service := range h.BinlogHandler.services {
		service.Start()
	}
	if h.Drive.Lock() {
		log.Debugf("current run as leader")
		h.StartService()
	}
}

func (h *Binlog) Reload(service string) {
	var (
		tcp  = "tcp"
		http = "http"
		all  = "all"
	)
	switch service {
	case tcp:
		log.Debugf("tcp service reload")
		h.BinlogHandler.services["tcp"].Reload()
	case http:
		log.Debugf("http service reload")
		h.BinlogHandler.services["http"].Reload()
	case all:
		log.Debugf("all service reload")
		h.BinlogHandler.services["tcp"].Reload()
		h.BinlogHandler.services["http"].Reload()
	}
}

func (h *Binlog) OnLeader() {
	log.Debugf("current run as leader, start running")
	h.StartService()
}
func (h *Binlog) OnPos(data []byte) {
	if data == nil {
		return
	}
	if len(data) < 19 {
		return
	}
	log.Debugf("onPos")
	// dl is file data length
	pos := int64(data[2]) | int64(data[3]) << 8 | int64(data[4]) << 16 |
		int64(data[5]) << 24 | int64(data[6]) << 32 | int64(data[7]) << 40 |
		int64(data[8]) << 48 | int64(data[9]) << 56
	eventIndex := int64(data[10]) | int64(data[11]) << 8 |
		int64(data[12]) << 16 | int64(data[13]) << 24 |
		int64(data[14]) << 32 | int64(data[15]) << 40 |
		int64(data[16]) << 48 | int64(data[17]) << 56
	log.Debugf("onPos %s, %d, %d", string(data[18:]), pos, eventIndex)
    h.BinlogHandler.lastBinFile = string(data[18:])
	h.BinlogHandler.lastPos = uint32(pos)
	h.BinlogHandler.EventIndex = eventIndex
	h.BinlogHandler.SaveBinlogPostionCache(string(data[18:]), pos, eventIndex)
}
