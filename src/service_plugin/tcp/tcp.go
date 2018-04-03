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
	if !ctx.TcpConfig.Enable{
		return &TcpService{status: 0}
	}
	tcp := &TcpService{
		Ip:               ctx.TcpConfig.Listen,
		Port:             ctx.TcpConfig.Port,
		lock:             new(sync.Mutex),
		statusLock:       new(sync.Mutex),
		groups:           make(map[string]*tcpGroup),
		wg:               new(sync.WaitGroup),
		listener:         nil,
		ctx:              ctx,
		ServiceIp:        ctx.TcpConfig.ServiceIp,
		status:           serviceEnable,
		token:            app.GetKey(app.CachePath + "/token"),
	}
	for _, group := range ctx.TcpConfig.Groups{
		tcpGroup := newTcpGroup(group)
		tcp.lock.Lock()
		tcp.groups.add(tcpGroup)
		tcp.lock.Unlock()
	}
	go tcp.keepalive()
	return tcp
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
	packData := pack(CMD_EVENT, data)
	// send to all groups
	for _, group := range tcp.groups {
		// check if match
		if group.match(table) {
			group.asyncSend(packData)
		}
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
	tcp.groups.asyncSend(msg)
	return true
}

func (tcp *TcpService) onClose(node *tcpClientNode) {
	tcp.lock.Lock()
	defer tcp.lock.Unlock()
	if node.status & tcpNodeIsNormal > 0 {
		if group, found := tcp.groups[node.group]; found {
			group.remove(node)
		}
		return
	}
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
			node := newNode(tcp.ctx, &conn, NodeClose(tcp.onClose), NodePro(tcp.setGroup))
			go node.onConnect()
		}
	}()
}

func (tcp *TcpService) setGroup(node *tcpClientNode, groupName string) bool {
	group, found := tcp.groups[groupName]
	if !found || groupName == "" {
		return false
	}
	group.append(node)
	return true
}

func (tcp *TcpService) Close() {
	log.Debugf("tcp service closing, waiting for buffer send complete.")
	tcp.lock.Lock()
	defer tcp.lock.Unlock()
	if tcp.listener != nil {
		(*tcp.listener).Close()
	}
	tcp.groups.close()
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
	// flag to mark if need restart
	restart := false
	// check if is need restart
	if tcp.Ip != tcp.ctx.TcpConfig.Listen || tcp.Port != tcp.ctx.TcpConfig.Port {
		log.Debugf("tcp service need to be restarted since ip address or/and port changed from %s:%d to %s:%d",
			tcp.Ip, tcp.Port, tcp.ctx.TcpConfig.Listen, tcp.ctx.TcpConfig.Port)
		restart = true
		// new config
		tcp.Ip = tcp.ctx.TcpConfig.Listen
		tcp.Port = tcp.ctx.TcpConfig.Port
		// close all connected nodes
		// remove all groups
		for _, group := range tcp.groups {
			group.nodes.close()
			tcp.lock.Lock()
			tcp.groups.delete(group)
			tcp.lock.Unlock()
		}
		// reset tcp config form new config
		for _, group := range tcp.ctx.TcpConfig.Groups { // new group
			tcpGroup := newTcpGroup(group)
			tcp.lock.Lock()
			tcp.groups.add(tcpGroup)
			tcp.lock.Unlock()
		}
	} else {
		// if listen ip or/and port does not change
		// 2-direction group comparision
		for name, group := range tcp.groups { // current group
			if !tcp.ctx.TcpConfig.Groups.HasName(name) {
				group.nodes.close()
				tcp.lock.Lock()
				tcp.groups.delete(group)
				tcp.lock.Unlock()
			} else {
				groupConfig := tcp.ctx.TcpConfig.Groups[name]
				tcp.groups[name].filter = groupConfig.Filter
			}
		}
		for _, group := range tcp.ctx.TcpConfig.Groups { // new group
			if tcp.groups.hasName(group.Name) {
				continue
			}
			// add it if new group found
			log.Debugf("new group: %s", group.Name)
			tcpGroup := newTcpGroup(group)
			tcp.lock.Lock()
			tcp.groups.add(tcpGroup)
			tcp.lock.Unlock()
		}
	}
	// if need restart, restart it
	if restart {
		log.Debugf("tcp service restart...")
		tcp.Close()
		tcp.Start()
	}
}

func (tcp *TcpService) keepalive() {
	for {
		select {
		case <-tcp.ctx.Ctx.Done():
			return
		default:
		}
		tcp.groups.asyncSend(packDataTickOk)
		time.Sleep(time.Second * 3)
	}
}

func (tcp *TcpService) Name() string {
	return "tcp"
}