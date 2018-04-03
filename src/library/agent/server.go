package agent

import (
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
	"library/app"
	"strings"
	"strconv"
	consul "github.com/hashicorp/consul/api"
)

//agent 所需要做的事情

//如果当前的节点不是leader
//那么查询leader的agent服务ip以及端口
//所有非leader节点连接到leader节点
//如果pos改变，广播到所有的非leader节点上
//非leader节点保存pos信息

// todo 这里还需要一个异常检测机制
// 定期检测是否有leader在运行，如果没有，尝试强制解锁，然后选出新的leader

const ServiceName = "wing-binlog-go-agent"

func NewAgentServer(ctx *app.Context, opts ...AgentServerOption) *TcpService {
	tcp := &TcpService{
		Address:          ctx.AppConfig.AgentfListen,
		lock:             new(sync.Mutex),
		statusLock:       new(sync.Mutex),
		wg:               new(sync.WaitGroup),
		listener:         nil,
		ctx:              ctx,
		agents:           nil,
		status:           0,
		buffer: make([]byte, 0),
	}
	go tcp.keepalive()
	tcp.client = newAgentClient(ctx)
	// 服务注册
	strs    := strings.Split(ctx.AppConfig.AgentfListen, ":")
	ip      := strs[0]
	port, _ := strconv.ParseInt(strs[1], 10, 32)
	conf := &consul.Config{Scheme: "http", Address: ctx.ClusterConfig.Consul.Address}
	c, err := consul.NewClient(conf)
	if err != nil {
		log.Panicf("%v", err)
		return nil
	}
	tcp.service = NewService(
		ctx.ClusterConfig.Lock,
		ServiceName,
		ip,
		int(port),
		c,
	)
	tcp.service.Register()
	if len(opts) > 0 {
		for _, f := range opts {
			f(tcp)
		}
	}
	OnLeader(tcp.client.OnLeader)(tcp)
	GetLeader(tcp.service.getLeader)(tcp.client)
	tcp.watch = newWatch(c, ServiceName, c.Health(), ip,
		int(port),onWatch(tcp.service.selectLeader),
		unlock(tcp.service.Unlock),
	)
	return tcp
}

// 设置收到pos的回调函数
func OnPos(f OnPosFunc) AgentServerOption  {
	return func(s *TcpService) {
		s.client.onPos = append(s.client.onPos, f)
	}
}

func OnLeader(f OnLeaderFunc) AgentServerOption {
	return func(s *TcpService) {
		s.service.onleader = append(s.service.onleader, f)
	}
}

// agent client 收到事件回调
// 这个回调应该来源于service_plugin/tcp
// 最终被转发到SendAll
func OnEvent(f OnEventFunc) AgentServerOption {
	return func(s *TcpService) {
		s.client.onEvent = append(s.client.onEvent, f)
	}
}

// agent client 收到一些其他的事件
// 原封不动转发到service_plugin/tcp SendRaw
func OnRaw(f OnRawFunc) AgentServerOption {
	return func(s *TcpService) {
		s.client.onRaw = append(s.client.onRaw, f)
	}
}

func (tcp *TcpService) Start() {
	go tcp.watch.process()
	go func() {
		listen, err := net.Listen("tcp", tcp.Address)
		if err != nil {
			log.Errorf("tcp service listen with error: %+v", err)
			return
		}
		tcp.listener = &listen
		log.Infof("agent service start with: %s", tcp.Address)
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
			node := newNode(tcp.ctx, &conn, NodeClose(tcp.agents.remove), NodePro(tcp.agents.append))
			go node.readMessage()
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
	tcp.agents.close()
	log.Debugf("tcp service closed.")
	tcp.service.Close()
}

// binlog的pos发生改变会通知到这里
// r为压缩过的二进制数据
// 可以直接写到pos cache缓存文件
func (tcp *TcpService) SendPos(data []byte) {
	if !tcp.service.leader {
		return
	}
	packData := pack(CMD_POS, data)
	tcp.agents.asyncSend(packData)
}

func (tcp *TcpService) SendEvent(table string, data []byte) {
	// 广播给agent client
	// agent client 再发送给连接到当前service_plugin/tcp的客户端
	packData := pack(CMD_EVENT, data)
	tcp.agents.asyncSend(packData)
}

// 心跳
func (tcp *TcpService) keepalive() {
	for {
		select {
		case <-tcp.ctx.Ctx.Done():
			return
		default:
		}
		tcp.agents.asyncSend(packDataTickOk)
		time.Sleep(time.Second * 3)
	}
}

func (tcp *TcpService) ShowMembers() string {
	return tcp.service.ShowMembers()
}
