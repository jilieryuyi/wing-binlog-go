package tcp

import (
	"net"
	"fmt"
	log "github.com/sirupsen/logrus"
	"sync"
	"context"
	"time"
)

type SyncClient struct {
	ctx               context.Context
	buffer            []byte
	bufferSize        int
	conn              net.Conn
	connLock          *sync.Mutex
	statusLock        *sync.Mutex
	status            int
	onMessageCallback []OnClientEventFunc
	asyncWriteChan    chan []byte
	ip                string
	port              int
	codec             ICodec
	onConnect         OnConnectFunc
	writeTimeout      time.Duration
	readTimeout       time.Duration
	connectTimeout    time.Duration
}

type SyncClientOption      func(tcp *SyncClient)

// 用来设置编码解码的接口
func SetSyncCoder(codec ICodec) SyncClientOption {
	return func(tcp *SyncClient) {
		tcp.codec = codec
	}
}

func SetSyncBufferSize(size int) SyncClientOption {
	return func(tcp *SyncClient) {
		tcp.bufferSize = size
	}
}

func SetWriteTimeout(t time.Duration) SyncClientOption {
	return func(tcp *SyncClient) {
		tcp.writeTimeout = t
	}
}

func SetReadTimeout(t time.Duration) SyncClientOption {
	return func(tcp *SyncClient) {
		tcp.readTimeout = t
	}
}

func SetConnectTimeout(t time.Duration) SyncClientOption {
	return func(tcp *SyncClient) {
		tcp.connectTimeout = t
	}
}

func NewSyncClient(ip string, port int, opts ...SyncClientOption) *SyncClient {
	c := &SyncClient{
		buffer:            make([]byte, 0),
		conn:              nil,
		statusLock:        new(sync.Mutex),
		status:            0,
		onMessageCallback: make([]OnClientEventFunc, 0),
		asyncWriteChan:    make(chan []byte, asyncWriteChanLen),
		connLock:          new(sync.Mutex),
		ip:                ip,
		port:              port,
		codec:             &Codec{},
		bufferSize:        4096,
		writeTimeout:      time.Duration(0),
		readTimeout:       time.Duration(0),
	}
	for _, f := range opts {
		f(c)
	}
	return c
}

// sync wait return
func (tcp *SyncClient) Send(data []byte) ([]byte, error) {
	if tcp.status & statusConnect <= 0 {
		return nil, NotConnect
	}
	if tcp.writeTimeout > 0 {
		tcp.conn.SetWriteDeadline(time.Now().Add(tcp.writeTimeout))
		defer tcp.conn.SetWriteDeadline(time.Time{})
	}
	sendMsg := tcp.codec.Encode(0, data)
	n, err := tcp.conn.Write(sendMsg)
	if n <= 0 || err != nil {
		return nil, err
	}
	//log.Infof("start read message %v", tcp.bufferSize)
	readBuffer := make([]byte, tcp.bufferSize)
	if tcp.readTimeout > 0 {
		tcp.conn.SetReadDeadline(time.Now().Add(tcp.readTimeout))
		defer tcp.conn.SetReadDeadline(time.Time{})
	}
	size, err  := tcp.conn.Read(readBuffer)
	if err != nil || size <= 0 {
		log.Warnf("client read with error: %+v", err)
		return nil, err
	}
	_, res, _, err :=tcp.codec.Decode(readBuffer[:size])
	return res, err
}

// use like go tcp.Connect()
func (tcp *SyncClient) Connect() error {
	if tcp.status & statusConnect > 0 {
		return IsConnected
	}
	d := net.Dialer{Timeout: tcp.connectTimeout}
	conn, err := d.Dial("tcp", fmt.Sprintf("%s:%d", tcp.ip, tcp.port))
	if err != nil {
		log.Errorf("start agent with error: %+v", err)
		return err
	}
	if tcp.status & statusConnect <= 0 {
		tcp.status |= statusConnect
	}
	tcp.conn = conn
	return nil
}

func (tcp *SyncClient) Disconnect() {
	if tcp.status & statusConnect <= 0 {
		return
	}
	tcp.conn.Close()
	if tcp.status & statusConnect > 0 {
		tcp.status ^= statusConnect
	}
}

