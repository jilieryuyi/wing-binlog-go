package binlog

import (
	"os"
	"sync"
	"library/services"
	"github.com/siddontang/go-mysql/canal"
	"library/app"
)

type Binlog struct {
	// github.com/siddontang/go-mysql interface
	canal.DummyEventHandler
	// config
	Config        *app.MysqlConfig
	// github.com/siddontang/go-mysql mysql protocol handler
	handler       *canal.Canal
	// context, like that use for wait coroutine exit
	ctx           *app.Context
	// use for wait coroutine exit
	wg            *sync.WaitGroup
	// lock
	lock          *sync.Mutex
	statusLock    *sync.Mutex
	// event unique index
	EventIndex    int64
	// registered service, key is the name of the service
	services      map[string]services.Service
	// cache handler, use for read and write cache file
	// binlog_handler.go SaveBinlogPostionCache and getBinlogPositionCache
	cacheHandler  *os.File
	// the last read pos
	lastPos       uint32
	// the last read binlog file
	lastBinFile   string

	startServiceChan chan struct{}
	stopServiceChan chan bool
	posChan chan []byte
	// binlog status
	status int

	//pos change 回调函数
	onPosChanges []PosChangeFunc
	onEvent []OnEventFunc
}

type BinlogOption func(h *Binlog)
type PosChangeFunc func(r []byte)
type OnEventFunc func(table string, data []byte)

const (
	//start stop
	_binlogIsRunning = 1 << iota
	// binlog is in exit status, will exit later
	_binlogIsExit
	_cacheHandlerIsOpened
)

const (
	posChanLen      = 10000
)