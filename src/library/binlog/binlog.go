package binlog

import (
	"github.com/siddontang/go-mysql/canal"
	"github.com/siddontang/go-mysql/mysql"
	"os"
	log "github.com/sirupsen/logrus"
	"library/file"
	"library/services"
	"library/buffer"
	"context"
	"sync"
	"fmt"
	"sync/atomic"
	"encoding/json"
	"time"
	"net"
)

func NewBinlog(ctx *context.Context) *Binlog {
	config, _ := GetMysqlConfig()
	var (
		b [defaultBufSize]byte
		err error
	)
	binlog := &Binlog{
		Config:   config,
		wg:       new(sync.WaitGroup),
		lock:     new(sync.Mutex),
		ctx:      ctx,
		isLeader: true,
		members:  make(map[string]*member),
	}
	cluster := NewCluster(ctx, binlog)
	cluster.Start()
	binlog.setMember(fmt.Sprintf("%s:%d", cluster.ServiceIp, cluster.port), true, 0)

	binlog.BinlogHandler = &binlogHandler{
		services:      make(map[string]services.Service),
		servicesCount: 0,
		lock:          new(sync.Mutex),
		Cluster:       cluster,
		ctx:           ctx,
	}
	f, p, index := binlog.BinlogHandler.getBinlogPositionCache()

	binlog.BinlogHandler.EventIndex = index
	binlog.BinlogHandler.buf = b[:0]
	binlog.BinlogHandler.isClosed = false

	binlog.isClosed = true

	binlog.handler = newCanal()
	binlog.handler.SetEventHandler(binlog.BinlogHandler)

	current_pos, err := binlog.handler.GetMasterPos()
	if f != "" {
		binlog.Config.BinFile = f
	} else {
		if err != nil {
			log.Panicf("binlog get cache error：%+v", err)
		} else {
			binlog.Config.BinFile = current_pos.Name
		}
	}
	if p > 0 {
		binlog.Config.BinPos = uint32(p)
	} else {
		if err != nil {
			log.Panicf("binlog get cache error：%+v", err)
		} else {
			binlog.Config.BinPos = current_pos.Pos
		}
	}

	binlog.BinlogHandler.lastBinFile = binlog.Config.BinFile
	binlog.BinlogHandler.lastPos = uint32(binlog.Config.BinPos)

	mysqlBinlogCacheFile := file.CurrentPath +"/cache/mysql_binlog_position.pos"
	dir := file.WPath{mysqlBinlogCacheFile}
	dir  = file.WPath{dir.GetParent()}
	dir.Mkdir()
	flag := os.O_WRONLY | os.O_CREATE | os.O_SYNC// | os.O_TRUNC
	binlog.BinlogHandler.cacheHandler, err = os.OpenFile(mysqlBinlogCacheFile, flag , 0755)
	if err != nil {
		log.Panicf("binlog open cache file error：%s, %+v", mysqlBinlogCacheFile, err)
	}
	return binlog
}

func newCanal() (*canal.Canal) {
	cfg, err := canal.NewConfigWithFile(file.CurrentPath + "/config/canal.toml")
	if err != nil {
		log.Panicf("binlog create canal config error：%+v", err)
	}
	handler, err := canal.NewCanal(cfg)
	if err != nil {
		log.Panicf("binlog create canal error：%+v", err)
	}
	return handler
}

// recover, try to rejoin to the cluster
func (h *Binlog) recover() {
	wfile := file.WFile{file.CurrentPath +"/cache/nodes.list"}
	str := wfile.ReadAll()
	if str == "" {
		log.Debug("recover empty")
		return
	}
	var nodes []interface{}
	err := json.Unmarshal([]byte(str), &nodes)
	if err != nil {
		log.Debugf("recover error with: %+v", err)
		return
	}
	log.Debugf("recover: %+v", nodes)
	dns := nodes[0].(string)
	log.Debugf("recover: %s", dns)
	h.BinlogHandler.Cluster.Client.ConnectTo(dns)
}

// set member
func (h *Binlog) setMember(dns string, isLeader bool, index int) int {
	log.Debugf("set member: %s, %t", dns, isLeader)
	h.lock.Lock()
	defer h.lock.Unlock()
	if index <= 0 {
		// get member index
		if len(h.members) == 0 {
			index = 1
		} else {
			for _, member := range h.members {
				if index <= 0 || index < member.index {
					index = member.index
				}
			}
			index++
		}
	}
	// check if exists
	mem, ok := h.members[dns]
	if ok {
		mem.isLeader = isLeader
		mem.index = index
	} else {
		h.members[dns] = &member{
			isLeader: isLeader,
			index:    index,
			status:   MEMBER_STATUS_LIVE,
		}
	}
	return index
}

// get leader dns
func (h *Binlog) getLeader() (string, int)  {
	h.lock.Lock()
	defer h.lock.Unlock()
	for dns, member := range h.members  {
		log.Debugf("getLeader: %s, %+v", dns, *member)
		if member.isLeader {
			return dns, member.index
		}
	}
	return "", 0
}

// set current isLeader
func (h *Binlog) leader(isLeader bool) {
	log.Debugf("binlog set leader %t", isLeader)
	h.lock.Lock()
	defer h.lock.Unlock()
	h.isLeader = isLeader
	dns := fmt.Sprintf("%s:%d", h.BinlogHandler.Cluster.ServiceIp, h.BinlogHandler.Cluster.port)

	if isLeader {
		for _, member := range h.members {
			member.isLeader = false
		}
	}

	if member, found := h.members[dns]; found {
		log.Debugf("%s is leader now", dns)
		member.isLeader = isLeader
	}
}

func (h *Binlog) setStatus(dns string, status string) {
	h.lock.Lock()
	defer h.lock.Unlock()
	if member, found := h.members[dns]; found {
		if member.status != status {
			log.Debugf("change %s status from %s to %s.", dns, member.status, status)
			member.status = status
		}
	}
}

func (h *Binlog) leaderDown() {
	for _, member := range h.members {
		if member.isLeader {
			member.isLeader = false
			member.status = MEMBER_STATUS_LEAVE
			break
		}
	}
}

func (h *Binlog) leaderChange() {
	currentDns := fmt.Sprintf("%s:%d", h.BinlogHandler.Cluster.ServiceIp, h.BinlogHandler.Cluster.port)

	for dns, member := range h.members {
		if member.status != MEMBER_STATUS_LEAVE && dns != currentDns {
			conn, err := net.DialTimeout("tcp", dns, time.Second*3)
			if err != nil {
				log.Errorf("connect to dns %s error: %+v", dns, err)
				return
			}
			buf := make([]byte, 256)
			sendMsg := h.BinlogHandler.Cluster.pack(CMD_LEADER_CHANGE, currentDns)
			conn.Write(sendMsg)
			conn.SetReadDeadline(time.Now().Add(time.Second*3))
			size, err := conn.Read(buf)
			if err != nil || size <= 0 {
				log.Errorf("send to dns %s error: %+v", dns, err)
				return
			}
			dataBuf := buffer.NewBuffer(TCP_RECV_DEFAULT_SIZE)
			dataBuf.Write(buf[:size])
			dataBuf.ReadInt32()
			cmd, err := dataBuf.ReadInt16() // 2字节 command
			log.Debugf("close confirm: %d", cmd)
			conn.Close()
			if cmd != CMD_LEADER_CHANGE {
				log.Debugf("leader change return error")
			}
		}
	}
}

func (h *Binlog) isNextLeader() bool {
	//todo 判断当前节点是否为下一个leader
	//current dns
	currentDns := fmt.Sprintf("%s:%d", h.BinlogHandler.Cluster.ServiceIp, h.BinlogHandler.Cluster.port)
	log.Debugf("current dns: %s", currentDns)
	leaderIndex := 0
	minIndex    := 0
	maxIndex    := 0
	for _, member := range h.members {
		if minIndex == 0 || member.index < minIndex {
			minIndex = member.index
		}
		if maxIndex == 0 || member.index > maxIndex {
			maxIndex = member.index
		}
		if member.isLeader {
			leaderIndex = member.index
		}
	}
	log.Debugf("leader index: %d", leaderIndex)
	// next leader index
	nextLeaderIndex := leaderIndex + 1
	if nextLeaderIndex > maxIndex {
		nextLeaderIndex = minIndex
	}
	log.Debugf("next leader index: %d", nextLeaderIndex)
	for dns, member := range h.members {
		// if next leader index == current index and dns == currentDns
		// so the current node is next leader
		// return true
		log.Debugf("%s -- %s", dns, currentDns)
		if member.index == nextLeaderIndex && dns == currentDns {
			log.Debugf("next leader is: %s", dns)
			return true
		}
	}
	return false
}
func (h *Binlog) getNextLeader() string {
	//todo 判断当前节点是否为下一个leader
	//current dns
	//currentDns := fmt.Sprintf("%s:%d", h.BinlogHandler.Cluster.ServiceIp, h.BinlogHandler.Cluster.port)
	leaderIndex := 0
	minIndex := 0
	maxIndex := 0
	for _, member := range h.members {
		if minIndex == 0 || member.index < minIndex {
			minIndex = member.index
		}
		if maxIndex == 0 || member.index > maxIndex {
			maxIndex = member.index
		}
		if member.isLeader {
			leaderIndex = member.index
		}
	}
	// next leader index
	nextLeaderIndex := leaderIndex + 1
	if nextLeaderIndex > maxIndex {
		nextLeaderIndex = minIndex
	}
	for dns, member := range h.members {
		// if next leader index == current index and dns == currentDns
		// so the current node is next leader
		// return true
		if member.index == nextLeaderIndex {
			return dns
		}
	}
	return ""
}

func (h *Binlog) ShowMembers() string {
	h.lock.Lock()
	defer h.lock.Unlock()
	l := len(h.members)
	res := fmt.Sprintf("cluster size: %d node(s)\r\n", l)
	res += fmt.Sprintf("======+=========================+==========+===============\r\n")
	res += fmt.Sprintf("%-6s| %-23s | %-8s | %s\r\n", "index", "node", "role", "status")
	res += fmt.Sprintf("------+-------------------------+----------+---------------\r\n")
	for dns, member := range h.members {
		role := "follower"
		if member.isLeader {
			role = "leader"
		}
		res += fmt.Sprintf("%-6d| %-23s | %-8s | %s\r\n", member.index, dns, role, member.status)
	}
	res += fmt.Sprintf("------+-------------------------+----------+---------------\r\n")
	return res
}

func (h *Binlog) Close() {
	log.Warn("binlog service exit")
	if h.isClosed  {
		return
	}
	h.lock.Lock()
	h.isClosed = true
	h.lock.Unlock()
	h.StopService(true)
	h.BinlogHandler.cacheHandler.Close()
	h.BinlogHandler.Cluster.Close()
	for name, service := range h.BinlogHandler.services {
		log.Debugf("%s service exit", name)
		service.Close()
		//log.Debugf("%s service exit end", name)
	}
}

func (h *Binlog) StopService(exit bool) {
	log.Debug("binlog service stop")
	h.BinlogHandler.lock.Lock()
	defer h.BinlogHandler.lock.Unlock()
	if !h.BinlogHandler.isClosed {
		h.handler.Close()
	} else {
		data := fmt.Sprintf("%s:%d:%d",
			h.BinlogHandler.lastBinFile,
			h.BinlogHandler.lastPos,
			atomic.LoadInt64(&h.BinlogHandler.EventIndex))
		h.BinlogHandler.SaveBinlogPostionCache(data)
	}
	h.BinlogHandler.isClosed = true
	h.isClosed = true
	//reset cacnal handler
	if !exit {
		h.handler = newCanal()
		h.handler.SetEventHandler(h.BinlogHandler)
	}
	//debug.PrintStack()
}

func (h *Binlog) StartService() {
	log.Debug("binlog start service")
	h.lock.Lock()
	defer h.lock.Unlock()
	if !h.isClosed {
		log.Debug("binlog service is not in close status")
		return
	}
	go func() {
		startPos := mysql.Position{
			Name: h.BinlogHandler.lastBinFile,
			Pos:  h.BinlogHandler.lastPos,
		}
		h.isClosed = false
		err := h.handler.RunFrom(startPos)
		if err != nil {
			if !h.isClosed && !h.BinlogHandler.isClosed {
				// 非关闭情况下退出
				log.Errorf("binlog service exit: %+v", err)
			} else {
				log.Warnf("binlog service exit: %+v", err)
			}
			return
		}
	}()
}

func (h *Binlog) Start() {
	for _, service := range h.BinlogHandler.services {
		service.Start()
	}
	h.StartService()
	go func() {
		time.Sleep(time.Microsecond*10)
		h.recover()
	}()
}

func (h *Binlog) Reload(service string) {
	var (
		tcp       = "tcp"
		websocket = "websocket"
		http      = "http"
		kafka     = "kafka"
		all       = "all"
	)
	switch service {
	case tcp:
		log.Debugf("tcp service reload")
		h.BinlogHandler.services["tcp"].Reload()
	case websocket:
		log.Debugf("websocket service reload")
		h.BinlogHandler.services["websocket"].Reload()
	case http:
		log.Debugf("http service reload")
		h.BinlogHandler.services["http"].Reload()
	case kafka:
		log.Debugf("kafka service reload")
	case all:
		log.Debugf("all service reload")
		h.BinlogHandler.services["tcp"].Reload()
		h.BinlogHandler.services["websocket"].Reload()
		h.BinlogHandler.services["http"].Reload()
	}
}

