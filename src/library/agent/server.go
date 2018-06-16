package agent

import (
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
	"library/app"
	"strings"
	"strconv"
	//consul "github.com/hashicorp/consul/api"
	"library/services"
	mconsul "github.com/jilieryuyi/wing-go/consul"
	mtcp "github.com/jilieryuyi/wing-go/tcp"
	"os"
	"fmt"
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
// 服务注册
const (
	Registered = 1 << iota
)
type OnLeaderFunc func(bool)
type TcpService struct {
	Address string               // 监听ip
	lock *sync.Mutex
	statusLock *sync.Mutex
	ctx *app.Context
	listener *net.Listener
	wg *sync.WaitGroup
	agents tcpClients
	status int
	conn *net.TCPConn
	buffer []byte
	//service *Service
	client *AgentClient
	//watch *ConsulWatcher
	enable bool
	sService mconsul.ILeader
	onleader []OnLeaderFunc
	leader bool
	server *mtcp.Server
}

func NewAgentServer(ctx *app.Context, opts ...AgentServerOption) *TcpService {
	config, _ := getConfig()
	if !config.Enable {
		s := &TcpService{
			enable: config.Enable,
		}
		for _, f := range opts {
			f(s)
		}
		return s
	}
	tcp := &TcpService{
		Address:          config.AgentListen,
		lock:             new(sync.Mutex),
		statusLock:       new(sync.Mutex),
		wg:               new(sync.WaitGroup),
		listener:         nil,
		ctx:              ctx,
		agents:           nil,
		status:           0,
		buffer:           make([]byte, 0),
		enable:           config.Enable,
		onleader:         make([]OnLeaderFunc, 0),
	}
	go tcp.keepalive()
	tcp.client = newAgentClient(ctx)
	// 服务注册
	strs    := strings.Split(config.AgentListen, ":")
	ip      := strs[0]
	port, _ := strconv.ParseInt(strs[1], 10, 32)

	//conf    := &consul.Config{Scheme: "http", Address: config.ConsulAddress}
	//c, err := consul.NewClient(conf)
	//if err != nil {
	//	log.Panicf("%v", err)
	//	return nil
	//}
	//tcp.service = NewService(
	//	config.Lock,
	//	ServiceName,
	//	ip,
	//	int(port),
	//	c,
	//)
	//tcp.service.Register()
	for _, f := range opts {
		f(tcp)
	}
	//将tcp.client.OnLeader注册到server的选leader回调
	OnLeader(tcp.client.OnLeader)(tcp)
	//将tcp.service.getLeader注册为client获取leader的api
	//GetLeader(tcp.service.getLeader)(tcp.client)
	//watch监听服务变化
	//tcp.watch = newWatch(
	//	c,
	//	ServiceName,
	//	c.Health(),
	//	ip,
	//	int(port),
	//	onWatch(tcp.service.selectLeader),
	//	unlock(tcp.service.Unlock),
	//)




	///////////////
	tcp.sService = mconsul.NewLeader(
		config.ConsulAddress,
		config.Lock,
		ServiceName,
		ip,
		int(port),
	)

	GetLeader(func() (string, int, error) {
		m, err := tcp.sService.Get()
		if err != nil {
			return "", 0, err
		}
		return m.ServiceIp, m.Port, nil
	})(tcp.client)


	tcp.server = mtcp.NewServer(ctx.Ctx, config.AgentListen, mtcp.SetOnServerMessage(tcp.onServerMessage))

	return tcp
}

// 设置收到pos的回调函数
func OnPos(f OnPosFunc) AgentServerOption  {
	return func(s *TcpService) {
		if !s.enable {
			return
		}
		s.client.onPos = append(s.client.onPos, f)
	}
}

func OnLeader(f OnLeaderFunc) AgentServerOption {
	return func(s *TcpService) {
		if !s.enable {
			f(true)
			return
		}
		s.onleader = append(s.onleader, f)
	}
}

// agent client 收到事件回调
// 这个回调应该来源于service_plugin/tcp
// 最终被转发到SendAll
func OnEvent(f OnEventFunc) AgentServerOption {
	return func(s *TcpService) {
		if !s.enable {
			return
		}
		s.client.onEvent = append(s.client.onEvent, f)
	}
}

// agent client 收到一些其他的事件
// 原封不动转发到service_plugin/tcp SendRaw
func OnRaw(f OnRawFunc) AgentServerOption {
	return func(s *TcpService) {
		if !s.enable {
			return
		}
		s.client.onRaw = append(s.client.onRaw, f)
	}
}

func (tcp *TcpService) onServerMessage(node *mtcp.ClientNode, msgId int64, data []byte) {
	//收到分组消息，加入分组
}

func (tcp *TcpService) Start() {
	if !tcp.enable {
		return
	}
	//go tcp.watch.process()
	/*go func() {
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
	}()*/
	tcp.server.Start()
	tcp.sService.Select(func(member *mconsul.ServiceMember) {
		tcp.leader = member.IsLeader
		for _, f := range tcp.onleader {
			f(member.IsLeader)
		}
	})
}

func (tcp *TcpService) Close() {
	if !tcp.enable {
		return
	}
	log.Debugf("tcp service closing, waiting for buffer send complete.")
	tcp.lock.Lock()
	defer tcp.lock.Unlock()
	if tcp.listener != nil {
		(*tcp.listener).Close()
	}
	tcp.agents.close()
	log.Debugf("tcp service closed.")
	//tcp.service.Close()

	tcp.server.Close()
	tcp.sService.Free()
}

// binlog的pos发生改变会通知到这里
// r为压缩过的二进制数据
// 可以直接写到pos cache缓存文件
func (tcp *TcpService) SendPos(data []byte) {
	if !tcp.enable {
		return
	}
	if !tcp.leader {
		return
	}
	//tcp.sService.
	packData := services.Pack(CMD_POS, data)
	tcp.agents.asyncSend(packData)
}

func (tcp *TcpService) SendEvent(table string, data []byte) {
	if !tcp.enable {
		return
	}
	// 广播给agent client
	// agent client 再发送给连接到当前service_plugin/tcp的客户端
	packData := services.Pack(CMD_EVENT, data)
	tcp.agents.asyncSend(packData)
}

// 心跳
func (tcp *TcpService) keepalive() {
	if !tcp.enable {
		return
	}
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
	if !tcp.enable {
		return "agent is not enable"
	}
	data, err := tcp.sService.GetServices(false)//.getMembers()
	if data == nil || err != nil {
		return ""
	}
	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
	}
	res := fmt.Sprintf("current node: %s(%s)\r\n", hostname, tcp.Address)
	res += fmt.Sprintf("cluster size: %d node(s)\r\n", len(data))
	res += fmt.Sprintf("======+=============================================+==========+===============\r\n")
	res += fmt.Sprintf("%-6s| %-43s | %-8s | %s\r\n", "index", "node", "role", "status")
	res += fmt.Sprintf("------+---------------------------------------------+----------+---------------\r\n")
	for i, member := range data {
		role := "follower"
		if member.IsLeader {
			role = "leader"
		}
		res += fmt.Sprintf("%-6d| %-43s | %-8s | %s\r\n", i, fmt.Sprintf("%s(%s:%d)", "", member.ServiceIp, member.Port), role, member.Status)
	}
	res += fmt.Sprintf("------+---------------------------------------------+----------+---------------\r\n")
	return res
}

