package services

import (
	"sync"
	"context"
	"time"
)

type HttpService struct {
	Service
	groups [][]*httpNode      // 客户端分组，现在支持两种分组，广播组合负载均衡组
	groupsMode []int          // 分组的模式 1，2 广播还是复载均衡
	groupsFilter [][]string   // 分组过滤器
	lock *sync.Mutex          // 互斥锁，修改资源时锁定
	sendFailureTimes int64    // 发送失败次数
	enable bool
	timeTick time.Duration    // 故障检测的时间间隔
	ctx *context.Context
	wg *sync.WaitGroup
	clientsCount int
}

type HttpConfig struct {
	Enable bool
	TimeTick time.Duration   //故障检测的时间间隔，单位为秒
	Groups map[string]httpNodeConfig
}

type httpNode struct {
	url string                // url
	sendQueue chan string     // 发送channel
	weight int                // 权重 0 - 100
	sendTimes int64           // 发送次数
	sendFailureTimes int64    // 发送失败次数
	isDown bool               // 是否因为故障下线的节点
	failureTimesFlag int32    // 发送失败次数，用于配合last_error_time检测故障，故障定义为：连续三次发生错误和返回错误
	lock *sync.Mutex          // 互斥锁，修改资源时锁定
	cache [][]byte
	cacheIndex int
	cacheIsInit bool
	cacheFull bool
}
