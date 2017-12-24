package services

import (
	log "github.com/sirupsen/logrus"
	"time"
	"sync/atomic"
	"sync"
	"strconv"
	"library/http"
	"regexp"
	"context"
	"runtime"
)

// 创建一个新的http服务
func NewHttpService(ctx *context.Context) *HttpService {
	config, _ := getHttpConfig()
	if !config.Enable {
		return &HttpService {
			enable:config.Enable,
		}
	}
	glen := len(config.Groups)
	client := &HttpService {
		lock               : new(sync.Mutex),
		groups             : make([][]*httpNode, glen),
		groupsMode         : make([]int, glen),
		groupsFilter       : make([][]string, glen),
		sendFailureTimes   : int64(0),
		enable             : config.Enable,
		timeTick           : config.TimeTick,
		wg                 : new(sync.WaitGroup),
		clientsCount       : 0,
		ctx                : ctx,
	}
	index := 0
	for _, v := range config.Groups {
		nodesLen := len(v.Nodes)
		client.groups[index]        = make([]*httpNode, nodesLen)
		client.groupsMode[index]   = v.Mode
		client.groupsFilter[index] = make([]string, len(v.Filter))
		client.groupsFilter[index] = append(client.groupsFilter[index][:0], v.Filter...)
		for i := 0; i < nodesLen; i++ {
			w, _ := strconv.Atoi(v.Nodes[i][1])
			client.groups[index][i] = &httpNode{
				url              : v.Nodes[i][0],
				weight           : w,
				sendQueue        : make(chan string, TCP_MAX_SEND_QUEUE),
				sendTimes        : int64(0),
				sendFailureTimes : int64(0),
				isDown           : false,
				lock             : new(sync.Mutex),
				failureTimesFlag : int32(0),
				cacheIsInit      : false,
			}
			client.clientsCount++
		}
		index++
	}
	return client
}

// 开始服务
func (client *HttpService) Start() {
	if !client.enable {
		return
	}
	if client.clientsCount > 0 {
		cpu := runtime.NumCPU()
		for _, clients := range client.groups {
			for _, h := range clients {
				go client.errorCheckService(h)
				// 启用cpu数量的服务协程
				for i := 0; i < cpu; i++ {
					client.wg.Add(1)
					go client.clientSendService(h)
				}
			}
		}
	}
}

func (client *HttpService) cacheInit(node *httpNode) {
	if node.cacheIsInit {
		return
	}
	node.cache = make([][]byte, HTTP_CACHE_LEN)
	for k := 0; k < HTTP_CACHE_LEN; k++ {
		node.cache[k] = nil//make([]byte, HTTP_CACHE_BUFFER_SIZE)
	}
	node.cacheIsInit  = true
	node.cacheIndex   = 0
	node.cacheFull    = false
}

func (client *HttpService) addCache(node *httpNode, msg []byte) {
	log.Debugf("http service add failure cache: %s", node.url)
	node.cache[node.cacheIndex] = append(node.cache[node.cacheIndex][:0], msg...)
	node.cacheIndex++
	if node.cacheIndex >= HTTP_CACHE_LEN {
		node.cacheIndex = 0
		node.cacheFull  = true
	}
}

func (client *HttpService) sendCache(node *httpNode) {
	if node.cacheIndex > 0 {
		log.Debugf("http service send failure cache: %s", node.url)
		if node.cacheFull {
			for j := node.cacheIndex; j < HTTP_CACHE_LEN; j++ {
				node.sendQueue <- string(node.cache[j])
			}
		}
		for j := 0; j < node.cacheIndex; j++ {
			node.sendQueue <- string(node.cache[j])
		}
		node.cacheFull  = false
		node.cacheIndex = 0
	}
}

// 节点故障检测与恢复服务
func (client *HttpService) errorCheckService(node *httpNode) {
	for {
		node.lock.Lock()
		if node.isDown {
			// 发送空包检测
			// post默认3秒超时，所以这里不会死锁
			log.Debugf("http服务-故障节点探测：%s", node.url)
			_, err := http.Post(node.url,[]byte{byte(0)})
			if err == nil {
				//重新上线
				node.isDown = false
				log.Warn("http服务节点恢复", node.url)
				//对失败的cache进行重发
				client.sendCache(node)
			} else {
				log.Errorf("http服务-故障节点发生错误：%+v", err)
			}
		}
		node.lock.Unlock()
		time.Sleep(time.Second * client.timeTick)
		select{
		case <-(*client.ctx).Done():
			log.Debugf("http服务errorCheckService退出：%s", node.url)
			return
		default:
		}
	}
}

// 节点服务协程
func (client *HttpService) clientSendService(node *httpNode) {
	defer client.wg.Done()
	for {
		select {
		case  msg, ok := <-node.sendQueue:
			if !ok {
				log.Warnf("http服务-发送消息channel通道关闭")
				return
			}
			if !node.isDown {
				atomic.AddInt64(&node.sendTimes, int64(1))
				log.Debug("http服务 post数据到url：",
					node.url, string(msg))
				data, err := http.Post(node.url, []byte(msg))
				if (err != nil) {
					atomic.AddInt64(&client.sendFailureTimes, int64(1))
					atomic.AddInt64(&node.sendFailureTimes, int64(1))
					atomic.AddInt32(&node.failureTimesFlag, int32(1))
					failure_times := atomic.LoadInt32(&node.failureTimesFlag)
					// 如果连续3次错误，标志位故障
					if failure_times >= 3 {
						//发生故障
						log.Warn(node.url, "http服务发生错误，下线节点", node.url)
						node.lock.Lock()
						node.isDown = true
						node.lock.Unlock()
					}
					log.Warn("http服务失败url和次数：", node.url, node.sendFailureTimes)
					client.cacheInit(node)
					client.addCache(node, []byte(msg))
				} else {
					node.lock.Lock()
					if node.isDown {
						node.isDown = false
					}
					node.lock.Unlock()
					failure_times := atomic.LoadInt32(&node.failureTimesFlag)
					//恢复即时清零故障计数
					if failure_times > 0 {
						atomic.StoreInt32(&node.failureTimesFlag, 0)
					}
					//对失败的cache进行重发
					client.sendCache(node)
				}
				log.Debug("http服务 post返回值：", node.url, string(data))
			} else {
				// 故障节点，缓存需要发送的数据
				// 这里就需要一个map[string][10000][]byte，最多缓存10000条
				// 保持最新的10000条
				client.addCache(node, []byte(msg))
			}
			case <-(*client.ctx).Done():
				if len(node.sendQueue) <= 0 {
					log.Debugf("http服务clientSendService退出：%s", node.url)
					return
				}
		}
	}
}

func (client *HttpService) SendAll(msg []byte) bool {
    if !client.enable {
        return false
    }
	client.lock.Lock()
	defer client.lock.Unlock()
	for index, clients := range client.groups {
		if len(clients) <= 0 {
			continue
		}
		mode      := client.groupsMode[index]
		filter    := client.groupsFilter[index]
		flen      := len(filter)
		tableLen  := int(msg[0]) + int(msg[1] << 8);
		table     := string(msg[2: tableLen + 2])
		if flen > 0 {
			isMatch := false
			for _, f := range filter {
				match, err := regexp.MatchString(f, table)
				if err != nil {
					continue
				}
				if match {
					isMatch = true
					break
				}
			}
			if !isMatch {
				continue
			}
		}
		// 如果不等于权重，即广播模式
		if mode != MODEL_WEIGHT {
			for _, conn := range clients {
				log.Debug("http send broadcast: %s=>%s", conn.url, string(msg[tableLen+2:]))
				if len(conn.sendQueue) >= cap(conn.sendQueue) {
					log.Warnf("http send buffer full(weight):%s, %s", conn.url, string(msg[tableLen+2:]))
					continue
				}
				conn.sendQueue <- string(msg[tableLen+2:])
			}
		} else {
			// 负载均衡模式
			// todo 根据已经sendTimes的次数负载均衡
			clen := len(clients)
			target := clients[0]
			//将发送次数/权重 作为负载基数，每次选择最小的发送
			js := float64(atomic.LoadInt64(&target.sendTimes))/float64(target.weight)
			for i := 1; i < clen; i++ {
				stimes := atomic.LoadInt64(&clients[i].sendTimes)
				if stimes == 0 {
					//优先发送没有发过的
					target = clients[i]
					break
				}
				njs := float64(stimes)/float64(clients[i].weight)
				if njs < js {
					js = njs
					target = clients[i]
				}
			}
			if len(target.sendQueue) < cap(target.sendQueue) {
				log.Debug("http send load balance msg: %s=>%s", target.url, string(msg[tableLen + 2:]))
				target.sendQueue <- string(msg[tableLen + 2:])
			} else {
				log.Warnf("http send buffer full(balance): %s, %s", target.url, string(msg[tableLen + 2:]))
			}
		}
	}
    return true
}

func (client *HttpService) Close() {
	log.Warn("http service close")
	if client.clientsCount > 0 {
		client.wg.Wait()
	}
}

func (tcp *HttpService) Reload() {
	config, _ := getHttpConfig()
	log.Debug("http service reload: %+v", config)
	tcp.enable = config.Enable
	tcp.clientsCount = 0
	for i, _ := range tcp.groups {
		tcp.groups[i] = make([]*httpNode, 0)
		tcp.groupsMode[i] = 0
		tcp.groupsFilter[i] = make([]string, 0)
	}
	index := 0
	for _, v := range config.Groups {
		nodesLen := len(v.Nodes)
		tcp.groups[index]        = make([]*httpNode, nodesLen)
		tcp.groupsMode[index]   = v.Mode
		tcp.groupsFilter[index] = make([]string, len(v.Filter))
		tcp.groupsFilter[index] = append(tcp.groupsFilter[index][:0], v.Filter...)
		for i := 0; i < nodesLen; i++ {
			w, _ := strconv.Atoi(v.Nodes[i][1])
			tcp.groups[index][i] = &httpNode{
				url                : v.Nodes[i][0],
				weight             : w,
				sendQueue         : make(chan string, TCP_MAX_SEND_QUEUE),
				sendTimes         : int64(0),
				sendFailureTimes : int64(0),
				isDown            : false,
				lock               : new(sync.Mutex),
				failureTimesFlag : int32(0),
				cacheIsInit      : false,
			}
			tcp.clientsCount++
		}
		index++
	}
}