package agent

import (
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
	"library/app"
	"strings"
	"strconv"
	//consul "github.com/hashicorp/consul/api"
	"library/service"
	mconsul "github.com/jilieryuyi/wing-go/consul"
	mtcp "github.com/jilieryuyi/wing-go/tcp"
	"os"
	"fmt"
	"time"
	//"encoding/json"
	"encoding/json"
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
type OnEventFunc       func(table string, data []byte) bool
type OnRawFunc         func(msg []byte) bool
type TcpService struct {
	Address string               // 监听ip
	lock *sync.Mutex
	statusLock *sync.Mutex
	ctx *app.Context
	listener *net.Listener
	wg *sync.WaitGroup
	//agents tcpClients
	status int
	conn *net.TCPConn
	buffer []byte
	//service *Service
	client *mtcp.Client//AgentClient
	//watch *ConsulWatcher
	enable bool
	sService mconsul.ILeader
	onLeader []OnLeaderFunc
	leader bool
	server *mtcp.Server
	onEvent []OnEventFunc
	onPos   []OnPosFunc
}

func NewAgentServer(ctx *app.Context, opts ...AgentServerOption) *TcpService {
	config, err := getConfig()
	if err != nil {
		log.Panicf("%v", err)
	}
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
		//agents:           nil,
		status:           0,
		buffer:           make([]byte, 0),
		enable:           config.Enable,
		//onleader:         make([]OnLeaderFunc, 0),
		onEvent:          make([]OnEventFunc, 0),
		onPos:            make([]OnPosFunc, 0),
	}
	tcp.client = mtcp.NewClient(ctx.Ctx, mtcp.SetOnMessage(tcp.onClientMessage))//newAgentClient(ctx)
	// 服务注册
	strs      := strings.Split(config.AgentListen, ":")
	ip        := strs[0]
	port, _   := strconv.ParseInt(strs[1], 10, 32)

	tcp.sService = mconsul.NewLeader(
		config.ConsulAddress,
		config.Lock,
		ServiceName,
		ip,
		int(port),
	)
	for _, f := range opts {
		f(tcp)
	}
	//OnLeader(tcp.connectToLeader)(tcp)
	//GetLeader(func() (string, int, error) {
	//	m, err := tcp.sService.Get()
	//	if err != nil {
	//		return "", 0, err
	//	}
	//	return m.ServiceIp, m.Port, nil
	//})(tcp.client)


	tcp.server = mtcp.NewServer(ctx.Ctx, config.AgentListen, mtcp.SetOnServerMessage(tcp.onServerMessage))

	return tcp
}

//func (tcp *TcpService) connectToLeader(isLeader bool) {
//	//
//}

// 设置收到pos的回调函数
func OnPos(f OnPosFunc) AgentServerOption  {
	return func(s *TcpService) {
		if !s.enable {
			return
		}
		s.onPos = append(s.onPos, f)
	}
}

func OnLeader(f OnLeaderFunc) AgentServerOption {
	return func(s *TcpService) {
		if !s.enable {
			f(true)
			return
		}
		s.onLeader = append(s.onLeader, f)
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
		s.onEvent = append(s.onEvent, f)
	}
}

// agent client 收到一些其他的事件
// 原封不动转发到service_plugin/tcp SendRaw
func OnRaw(f OnRawFunc) AgentServerOption {
	return func(s *TcpService) {
		if !s.enable {
			return
		}
		//s.client.onRaw = append(s.client.onRaw, f)
	}
}

func (tcp *TcpService) onClientMessage(client *mtcp.Client, content []byte) {
	cmd, data, err := service.Unpack(content)
	if err != nil {
		log.Error(err)
		return
	}
	switch cmd {
	case CMD_EVENT:
		var raw map[string] interface{}
		err = json.Unmarshal(data, &raw)
		if err == nil {
			table := raw["database"].(string) + "." + raw["table"].(string)
			for _, f := range tcp.onEvent  {
				f(table, data)
			}
		}
	case CMD_POS:
		for _, f := range tcp.onPos  {
			f(data)
		}
	}
}

func (tcp *TcpService) onServerMessage(node *mtcp.ClientNode, msgId int64, data []byte) {

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
		log.Infof("current node %v is leader: %v", tcp.Address, member.IsLeader)
		tcp.leader = member.IsLeader
		for _, f := range tcp.onLeader {
			f(member.IsLeader)
		}
		// 连接到leader
		if !tcp.leader {
			for {
				m, err := tcp.sService.Get()
				if err == nil && m != nil {
					leaderAddress := fmt.Sprintf("%v:%v", m.ServiceIp, m.Port)
					log.Infof("connect to leader %v", leaderAddress)
					tcp.client.Connect(leaderAddress, time.Second * 3)
					break
				}
				log.Warnf("leader is not init, try to wait init")
				time.Sleep(time.Second)
			}
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
	//tcp.agents.close()
	log.Debugf("tcp service closed.")
	//tcp.service.Close()

	tcp.server.Close()
	tcp.sService.Free()
}

// 此api提供给binlog通过agent server同步广播发送给所有的client客户端
func (tcp *TcpService) Sync(data []byte) {
	if !tcp.enable {
		return
	}
	// 广播给agent client
	// agent client 再发送给连接到当前service_plugin/tcp的客户端
	//packData := service.Pack(CMD_EVENT, data)
	tcp.server.Broadcast(1, data)
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

