package tcp

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
	"library/app"
	"library/services"
)

func NewTcpService(ctx *app.Context) services.Service {
	g := newGroups(ctx)
	t := newTcpService(ctx,
		SetSendAll(g.sendAll),
		SetSendRaw(g.asyncSend),
		SetOnConnect(g.onConnect),
		SetOnClose(g.close),
		SetKeepalive(g.asyncSend),
		SetReload(g.reload),
	)
	return t
}

func newTcpService(ctx *app.Context, opts ...TcpServiceOption) services.Service {
	tcp := &TcpService{
		Ip:               ctx.TcpConfig.Listen,
		Port:             ctx.TcpConfig.Port,
		lock:             new(sync.Mutex),
		statusLock:       new(sync.Mutex),
		wg:               new(sync.WaitGroup),
		listener:         nil,
		ctx:              ctx,
		ServiceIp:        ctx.TcpConfig.ServiceIp,
		status:           serviceEnable,
		token:            app.GetKey(app.CachePath + "/token"),
		sendAll:          make([]SendAllFunc, 0),
		sendRaw:          make([]SendRawFunc, 0),
		onConnect:        make([]OnConnectFunc, 0),
		onClose:          make([]CloseFunc, 0),
		onKeepalive:      make([]KeepaliveFunc, 0),
		reload:           make([]ReloadFunc, 0),
	}
	for _, f := range opts {
		f(tcp)
	}
	go tcp.keepalive()
	log.Debugf("-----tcp service init----")
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
	log.Debugf("tcp SendAll: %s, %+v", table, string(data))
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
	//tcp.groups.asyncSend(msg)

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
	tcp.statusLock.Unlock()
	go func() {
		dns := fmt.Sprintf("%s:%d", tcp.Ip, tcp.Port)
		listen, err := net.Listen("tcp", dns)
		if err != nil {
			log.Errorf("tcp service listen with error: %+v", err)
			return
		}
		tcp.listener = &listen
		log.Infof("tcp service start with: %s", dns)
		for {
			conn, err := listen.Accept()
			select {
			case <-tcp.ctx.Ctx.Done():
				return
			default:
			}
			if err != nil {
				log.Warnf("tcp service accept with error: %+v", err)
				continue
			}
			//node := newNode(tcp.ctx, &conn, NodeClose(tcp.groups.removeNode), NodePro(tcp.groups.addNode))
			//go node.onConnect()

			for _, f := range tcp.onConnect {
				f(&conn)
			}
		}
	}()
}

func (tcp *TcpService) Close() {
	log.Debugf("tcp service closing, waiting for buffer send complete.")
	tcp.lock.Lock()
	defer tcp.lock.Unlock()
	if tcp.listener != nil {
		(*tcp.listener).Close()
	}
	//tcp.groups.close()
	for _, f := range tcp.onClose {
		f()
	}
	log.Debugf("tcp service closed.")
}

func (tcp *TcpService) Reload() {
	tcp.ctx.ReloadTcpConfig()
	log.Debugf("tcp service reload with new config：%+v", tcp.ctx.TcpConfig)
	tcp.statusLock.Lock()
	if tcp.ctx.TcpConfig.Enable && tcp.status & serviceEnable <= 0 {
		tcp.status |= serviceEnable
	}
	if !tcp.ctx.TcpConfig.Enable && tcp.status & serviceEnable > 0 {
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
		//tcp.groups.asyncSend(packDataTickOk)
		for _, f := range tcp.onKeepalive {
			f(packDataTickOk)
		}
		time.Sleep(time.Second * 3)
	}
}

func (tcp *TcpService) Name() string {
	return "tcp"
}