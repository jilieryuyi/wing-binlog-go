package subscribe

import (
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
	"library/app"
	"library/services"
	"strings"
	"strconv"
)

func NewSubscribeService(ctx *app.Context) *TcpService {
	config, _ := getConfig()

	g := newGroups(ctx)
	t := newSubscribeService(
		ctx,
		config.Enable,
		config.Listen,
		SetSendAll(g.sendAll),
		SetSendRaw(g.asyncSend),
		SetOnConnect(g.onConnect),
		SetOnClose(g.close),
		SetKeepalive(g.asyncSend),
		SetReload(g.reload),
	)

	// 服务注册相关
	if config.ConsulEnable && config.ConsulAddress != "" {
		temp := strings.Split(config.Listen, ":")
		log.Debugf("listen: %v", config.Listen)
		host    := temp[0]
		port, _ := strconv.ParseInt(temp[1], 10, 32)
		log.Debugf("%v==%v", host, port)
		service := NewService(host, int(port), config.ConsulAddress)
		service.Register()
		//注入close依赖
		SetOnClose(service.Close)(t)
		SetOnConnect(service.newConnect)(t)
		SetOnRemove(service.disconnect)(g)
	}

	return t
}

func newSubscribeService(
	ctx *app.Context,
	enable bool,
	listen string,
	opts ...TcpServiceOption) *TcpService {

	tcp := &TcpService{
		Enable:           enable,//config.Enable,
		Listen:           listen,//config.Listen,
		lock:             new(sync.Mutex),
		statusLock:       new(sync.Mutex),
		wg:               new(sync.WaitGroup),
		listener:         nil,
		ctx:              ctx,
		status:           0,
		sendAll:          make([]SendAllFunc, 0),
		sendRaw:          make([]SendRawFunc, 0),
		onConnect:        make([]OnConnectFunc, 0),
		onClose:          make([]CloseFunc, 0),
		onKeepalive:      make([]KeepaliveFunc, 0),
		reload:           make([]ReloadFunc, 0),
	}
	if enable {
		tcp.status |= serviceEnable
	}
	for _, f := range opts {
		f(tcp)
	}
	go tcp.keepalive()
	log.Debugf("-----subscribe service init----")
	return tcp
}

func SetSendAll(f SendAllFunc) TcpServiceOption {
	return func(service *TcpService) {
		service.sendAll = append(service.sendAll, f)
	}
}

func SetSendRaw(f SendRawFunc) TcpServiceOption {
	return func(service *TcpService) {
		service.sendRaw = append(service.sendRaw, f)
	}
}

func SetOnConnect(f OnConnectFunc) TcpServiceOption {
	return func(service *TcpService) {
		service.onConnect = append(service.onConnect, f)
	}
}

func SetOnClose(f CloseFunc) TcpServiceOption{
	return func(service *TcpService) {
		service.onClose = append(service.onClose, f)
	}
}

func SetKeepalive(f KeepaliveFunc) TcpServiceOption{
	return func(service *TcpService) {
		service.onKeepalive = append(service.onKeepalive, f)
	}
}

func SetReload(f ReloadFunc) TcpServiceOption {
	return func(service *TcpService) {
		service.reload = append(service.reload, f)
	}
}

// send event data to all connects client
func (tcp *TcpService) SendAll(table string, data []byte) bool {
	tcp.statusLock.Lock()
	if tcp.status & serviceEnable <= 0 {
		tcp.statusLock.Unlock()
		return false
	}
	tcp.statusLock.Unlock()
	log.Debugf("subscribe SendAll: %s, %+v", table, string(data))
	// pack data
	packData := services.Pack(CMD_EVENT, data)
	for _, f := range tcp.sendAll {
		f(table, packData)
	}
	return true
}

// send raw bytes data to all connects client
// msg is the pack frame form func: pack
func (tcp *TcpService) SendRaw(msg []byte) bool {
	tcp.statusLock.Lock()
	if tcp.status & serviceEnable <= 0 {
		tcp.statusLock.Unlock()
		return false
	}
	tcp.statusLock.Unlock()
	log.Debugf("tcp sendRaw: %+v", msg)
	for _, f := range tcp.sendRaw {
		f(msg)
	}
	return true
}

func (tcp *TcpService) Start() {
	tcp.statusLock.Lock()
	if tcp.status & serviceEnable <= 0 {
		tcp.statusLock.Unlock()
		return
	}
	if tcp.status & serviceClosed > 0 {
		tcp.status ^= serviceClosed
	}
	tcp.statusLock.Unlock()
	go func() {
		listen, err := net.Listen("tcp", tcp.Listen)
		if err != nil {
			log.Errorf("tcp service listen with error: %+v", err)
			return
		}
		tcp.listener = &listen
		log.Infof("tcp service start with: %s", tcp.Listen)
		for {
			conn, err := listen.Accept()
			select {
			case <-tcp.ctx.Ctx.Done():
				return
			default:
			}
			tcp.statusLock.Lock()
			if tcp.status & serviceClosed > 0 {
				tcp.statusLock.Unlock()
				return
			}
			tcp.statusLock.Unlock()
			if err != nil {
				log.Warnf("tcp service accept with error: %+v", err)
				continue
			}
			for _, f := range tcp.onConnect {
				f(&conn)
			}
		}
	}()
}

func (tcp *TcpService) Close() {
	if tcp.status & serviceClosed > 0 {
		return
	}
	log.Debugf("tcp service closing, waiting for buffer send complete.")
	tcp.lock.Lock()
	defer tcp.lock.Unlock()
	if tcp.listener != nil {
		(*tcp.listener).Close()
	}
	for _, f := range tcp.onClose {
		f()
	}
	if tcp.status & serviceClosed <= 0 {
		tcp.status |= serviceClosed
	}
	log.Debugf("tcp service closed.")
}

func (tcp *TcpService) Reload() {
	config, _ := getConfig()
	log.Debugf("tcp service reload with new config：%+v", *config)
	tcp.statusLock.Lock()
	if config.Enable && tcp.status & serviceEnable <= 0 {
		tcp.status |= serviceEnable
	}
	if !config.Enable && tcp.status & serviceEnable > 0 {
		tcp.status ^= serviceEnable
	}
	tcp.statusLock.Unlock()
	for _, f := range tcp.reload {
		f()
	}
	log.Debugf("tcp service restart...")
	tcp.Close()
	tcp.Start()
}

func (tcp *TcpService) keepalive() {
	if tcp.status & serviceEnable <= 0 {
		return
	}
	for {
		select {
		case <-tcp.ctx.Ctx.Done():
			return
		default:
		}
		for _, f := range tcp.onKeepalive {
			f(packDataTickOk)
		}
		time.Sleep(time.Second * 3)
	}
}

func (tcp *TcpService) Name() string {
	return "subscribe"
}