package binlog

import (
	"github.com/siddontang/go-mysql/canal"
	"github.com/siddontang/go-mysql/mysql"
	"os"
	"strings"
	"sync/atomic"
	"fmt"
	log "github.com/sirupsen/logrus"
	"library/file"
	"library/services"
	wstring "library/string"
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
	f, p, index := binlog.getBinlogPositionCache()
	var b [defaultBufSize]byte
	binlog.BinlogHandler = binlogHandler{
		Event_index: index,
		services:make([]services.Service, 4),
		services_count:0,
	}
	binlog.BinlogHandler.buf = b[:0]
	binlog.BinlogHandler.chan_save_position = make(chan positionCache, MAX_CHAN_FOR_SAVE_POSITION)
	binlog.handler.SetEventHandler(&binlog.BinlogHandler)
	binlog.is_connected = false
	if f != "" {
		binlog.Config.BinFile = f
	}
	if p > 0 {
		binlog.Config.BinPos = p
	}
	log.Debugf("%+v", binlog.Config)
	return binlog
}

func (h *Binlog) Close() {
	if !h.is_connected  {
		return
	}
	h.handler.Close()
	h.is_connected = false
	close(h.BinlogHandler.chan_save_position)
	for _, service := range h.BinlogHandler.services {
		service.Close()
	}
}

func (h *binlogHandler) SaveBinlogPostionCache(p mysql.Position) {
	if len(h.chan_save_position) >= cap(h.chan_save_position) {
		log.Warn("binlgo服务-SaveBinlogPostionCache缓冲区满...")
		return
	}
	h.chan_save_position <- positionCache{p, atomic.LoadInt64(&h.Event_index)}
}

func (h *Binlog) getBinlogPositionCache() (string, int64, int64) {
	wfile := file.WFile{file.GetCurrentPath() +"/cache/mysql_binlog_position.pos"}
	str := wfile.ReadAll()
	if str == "" {
		return "", int64(0), int64(0)
	}
	res := strings.Split(str, ":")
	if len(res) < 3 {
		return "", int64(0), int64(0)
	}
	wstr  := wstring.WString{res[1]}
	pos   := wstr.ToInt64()
	wstr2 := wstring.WString{res[2]}
	index := wstr2.ToInt64()
	return res[0], pos, index
}

func (h *Binlog) writeCache() {
	wfile := file.WFile{file.GetCurrentPath() +"/cache/mysql_binlog_position.pos"}
	for {
		select {
		case pos := <-h.BinlogHandler.chan_save_position:
			if pos.pos.Name != "" && pos.pos.Pos > 0 {
				wfile.Write(fmt.Sprintf("%s:%d:%d", pos.pos.Name, pos.pos.Pos, pos.index), false)
			}
		}
	}
}

func (h *Binlog) Start() {
	for _, service := range h.BinlogHandler.services {
		service.Start()
	}
	log.Debugf("binlog调试：%s,%d", h.Config.BinFile, uint32(h.Config.BinPos))
	go h.writeCache()
	go func() {
		startPos := mysql.Position{
			Name: h.Config.BinFile,
			Pos:  uint32(h.Config.BinPos),
		}
		err := h.handler.RunFrom(startPos)
		if err != nil {
			log.Fatalf("binlog服务：start canal err %v", err)
			return
		}
		h.is_connected = true
	}()
}
