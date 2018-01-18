package cluster

import (
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type Mysql struct{
	IsLeader bool
	lock *sync.Mutex
}

const (
	LOCK = "wing-binlog-cluster-lock"
	TIMEOUT = 6 //6秒超时
)

/**
CREATE TABLE `lock` (
 `ip` varchar(128) NOT NULL DEFAULT '' COMMENT '成功获取锁的ip',
 `lock` varchar(128) NOT NULL DEFAULT '' COMMENT '锁',
 `updated` int(11) NOT NULL DEFAULT '0' COMMENT '最后的更新时间，用于keepalive',
 UNIQUE KEY `lock` (`lock`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8
*/

// 校验当前节点是否为leader，返回true，说明是leader，则当前节点开始工作
// 返回false，说明是follower
func (d *Mysql) Leader() bool {
	sqlStr := "INSERT INTO `lock`(`ip`, `lock`, `updated`) VALUES (?,?,?)"
	log.Debugf("%s", sqlStr)

	//如果获取锁成功
	d.lock.Lock()
	d.IsLeader = true
	d.lock.Unlock()
	return true
}

// 同步游标相关信息
func (d *Mysql) Write(binFile string, pos int64, eventIndex int64) bool {
	return true
}

// 读取游标信息
// 返回binFile string, pos int64, eventIndex int64
func (d *Mysql) Read() (string, int64, int64) {
	return "", 0, 0
}

// 获取当前的集群成员
func (d *Mysql) Members() []ClusterMember {
	return nil
}

func (d *Mysql) keepAlive() {
	for {
		d.lock.Lock()
		if d.IsLeader {
			//leader节点保持心跳
			//更新updated为当前时间戳
		}
		d.lock.Unlock()
		time.Sleep(time.Second * 2)
	}
}

func (d *Mysql) checkTimeout() {
	for {
		d.lock.Lock()
		if !d.IsLeader {
			//非leader节点判断超时
			//读取updated字段与当前时间比对
			//如果超过TIMEOUT，ze重新选leader
		}
		d.lock.Unlock()
		time.Sleep(time.Second * 2)
	}
}

func (d *Mysql) Start() {
	go d.keepAlive()
	go d.checkTimeout()
}

