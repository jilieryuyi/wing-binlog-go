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
)

func NewBinlog(ctx *context.Context) *Binlog {
	config, _ := GetMysqlConfig()
	debug_config := config
	debug_config.Password = "******"
	log.Debugf("binlog配置：%+v", debug_config)
	binlog := &Binlog {
		Config : config,
		wg : new(sync.WaitGroup),
		lock : new(sync.Mutex),
		ctx : ctx,
	}
	config_file := file.GetCurrentPath() + "/config/canal.toml"
	cfg, err := canal.NewConfigWithFile(config_file)
	if err != nil {
		log.Panic("binlog错误：", err)
		os.Exit(1)
	}
	debug_cfg := *cfg
	debug_cfg.Password = "******"
	log.Debugf("binlog配置(cfg)：%+v", debug_cfg)
	binlog.handler, err = canal.NewCanal(cfg)
	if err != nil {
		log.Panicf("binlog创建canal错误：%+v", err)
		os.Exit(1)
	}
	var b [defaultBufSize]byte

	// 集群服务
	cluster := NewCluster(ctx, binlog)
	cluster.Start()

	binlog.BinlogHandler = &binlogHandler{
		services : make(map[string]services.Service),
		servicesCount : 0,
		lock : new(sync.Mutex),
		wg : new(sync.WaitGroup),
		Cluster : cluster,
		ctx : ctx,
	}
	binlog.BinlogHandler.wg.Add(1)
	f, p, index := binlog.BinlogHandler.getBinlogPositionCache()
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
			log.Panicf("binlog获取GetMasterPos错误：%+v", err)
		} else {
			binlog.Config.BinFile = current_pos.Name
		}
	}
	if p > 0 {
		binlog.Config.BinPos = p
	} else {
		if err != nil {
			log.Panicf("binlog获取GetMasterPos错误：%+v", err)
		} else {
			binlog.Config.BinPos = int64(current_pos.Pos)
		}
	}
	log.Debugf("binlog配置：%+v", binlog.Config)

	binlog.BinlogHandler.lastBinFile = binlog.Config.BinFile
	binlog.BinlogHandler.lastPos = uint32(binlog.Config.BinPos)

	// 初始化缓存文件句柄
	mysql_binlog_position_cache := file.GetCurrentPath() +"/cache/mysql_binlog_position.pos"
	dir := file.WPath{mysql_binlog_position_cache}
	dir = file.WPath{dir.GetParent()}
	dir.Mkdir()
	flag := os.O_WRONLY | os.O_CREATE | os.O_SYNC | os.O_TRUNC
	binlog.BinlogHandler.cacheHandler, err = os.OpenFile(
		mysql_binlog_position_cache, flag , 0755)
	if err != nil {
		log.Panicf("binlog服务，打开缓存文件错误：%s, %+v", mysql_binlog_position_cache, err)
	}
	return binlog
}

func (h *Binlog) Close() {
	log.Debug("binlog服务退出...")
	if h.isClosed  {
		return
	}
	h.BinlogHandler.lock.Lock()
	h.isClosed = true
	// 退出顺序，先停止canal对mysql数据的接收
	if !h.BinlogHandler.isClosed {
		h.handler.Close()
	}
	h.BinlogHandler.isClosed = true
	log.Debug("binlog-h.handler.Close退出...")
	h.BinlogHandler.lock.Unlock()

	//等待cache写完
	log.Debug("binlog-h.BinlogHandler.cacheHandler 等待退出...")
	h.BinlogHandler.wg.Wait()
	//关闭cache
	h.BinlogHandler.cacheHandler.Close()
	h.BinlogHandler.Cluster.Close()
	log.Debug("binlog-h.BinlogHandler.cacheHandler.Close退出...")

	//以下操作服务内部完成
	//等待服务数据发送完成
	//关闭服务
	for _, service := range h.BinlogHandler.services {
		log.Debug("服务退出...")
		service.Close()
	}
	log.Debug("binlog-服务Close-all退出...")
}

// 仅仅停止服务
func (h *Binlog) StopService() {
	log.Debug("停止binlog服务")
	h.BinlogHandler.lock.Lock()
	h.BinlogHandler.isClosed = true
	// 退出顺序，先停止canal对mysql数据的接收
	h.handler.Close()
	h.BinlogHandler.lock.Unlock()
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
				log.Errorf("wing-binlog-go service exit: %+v", err)
			} else {
				log.Infof("wing-binlog-go service exit: %+v", err)
			}
			return
		}
	}()
}

func (h *Binlog) Start() {
	for _, service := range h.BinlogHandler.services {
		service.Start()
		service.SetContext(h.ctx)
	}
	log.Debugf("binlog调试：%s,%d", h.Config.BinFile, uint32(h.Config.BinPos))
	h.StartService()
}

func (h *Binlog) Reload(service string) {
	var (
		tcp = "tcp"
		websocket = "websocket"
		http = "http"
		kafka = "kafka"
		all = "all"
	)
	switch service {
	case tcp:
		log.Debugf("重新加载tcp服务")
		h.BinlogHandler.services["tcp"].Reload()
	case websocket:
		log.Debugf("重新加载websocket服务")
		h.BinlogHandler.services["websocket"].Reload()
	case http:
		log.Debugf("重新加载http服务")
		h.BinlogHandler.services["http"].Reload()
	case kafka:
		log.Debugf("重新加载kafka服务")
	case all:
		log.Debugf("重新加载全部服务")
		h.BinlogHandler.services["tcp"].Reload()
		h.BinlogHandler.services["websocket"].Reload()
		h.BinlogHandler.services["http"].Reload()
	}
}
