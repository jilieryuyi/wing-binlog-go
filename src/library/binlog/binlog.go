package binlog

import (
	"github.com/siddontang/go-mysql/canal"
	"github.com/siddontang/go-mysql/mysql"
	"os"
	log "github.com/sirupsen/logrus"
	"library/file"
	"library/services"
	"context"
)

func NewBinlog() *Binlog {
	config, _ := GetMysqlConfig()
	debug_config := config
	debug_config.Password = "******"
	log.Debugf("binlog配置：%+v", debug_config)
	binlog := &Binlog {
		Config:config,
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
	f, p, index := binlog.BinlogHandler.getBinlogPositionCache()
	var b [defaultBufSize]byte
	binlog.BinlogHandler = binlogHandler{
		Event_index: index,
		services:make([]services.Service, 4),
		services_count:0,
	}
	binlog.BinlogHandler.buf = b[:0]
	binlog.handler.SetEventHandler(&binlog.BinlogHandler)
	binlog.is_connected = false
	if f != "" {
		binlog.Config.BinFile = f
	}
	if p > 0 {
		binlog.Config.BinPos = p
	}
	log.Debugf("%+v", binlog.Config)

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
	if !h.is_connected  {
		return
	}
	h.is_connected = false
	go func() {
		for _, service := range h.BinlogHandler.services {
			log.Debug("服务退出...")
			service.Close()
		}
		h.BinlogHandler.cacheHandler.Close()
	}()
	h.handler.Close()
}


func (h *Binlog) Start(ctx *context.Context) {
	h.ctx = ctx
	for _, service := range h.BinlogHandler.services {
		service.Start()
		service.SetContext(ctx)
	}
	log.Debugf("binlog调试：%s,%d", h.Config.BinFile, uint32(h.Config.BinPos))
	go func() {
		startPos := mysql.Position{
			Name: h.Config.BinFile,
			Pos:  uint32(h.Config.BinPos),
		}
		h.is_connected = true
		err := h.handler.RunFrom(startPos)
		if err != nil {
			log.Fatalf("binlog服务：start canal err %v", err)
			return
		}
	}()
}
