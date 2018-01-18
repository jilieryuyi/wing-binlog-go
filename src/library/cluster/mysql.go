package cluster

import (
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
	"database/sql"
	"fmt"
)

type Mysql struct{
	IsLeader bool
	lock *sync.Mutex

	addr string
	port int
	user string
	password string
	database string
	charset string
	handler *sql.DB
}

const (
	LOCK = "wing-binlog-cluster-lock"
	TIMEOUT = 6 //6秒超时
)

func NewMysql() *Mysql {
	config, _ := GetConfig()
	m := &Mysql{
		addr: config.Mysql.Addr,
		port: config.Mysql.Port,
		user: config.Mysql.User,
		password: config.Mysql.Password,
		database: config.Mysql.Database,
		charset: config.Mysql.Charset,
	}
	m.init()
	return m
}

func (m *Mysql) init() {
	dataSource := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=%s",
		m.user, m.password, m.addr,
		m.port, m.database, m.charset)
	var err error
	m.handler, err = sql.Open("mysql", dataSource)
	if err != nil {
		log.Errorf("connect database with error：%+v", err)
		m.handler = nil
		return
	}
	m.handler.SetMaxIdleConns(8)
	m.handler.SetMaxOpenConns(16)

	sqlStr := "select bin_file from `cursor`"
	row := m.handler.QueryRow(sqlStr)
	var (
		binFile string
	)
	err = row.Scan(&binFile)
	if err != nil {
		log.Errorf("query sql with error：%s, %+v", sqlStr, err)
		// init data
		sqlStr = "INSERT INTO `cursor`(`bin_file`, `pos`, `event_index`, `updated`) VALUES (?,?,?,?)"
		m.handler.Exec(sqlStr, "", 0, 0, time.Now().Unix())
	}
}

/**
CREATE TABLE `lock` (
 `ip` varchar(128) NOT NULL DEFAULT '' COMMENT '成功获取锁的ip',
 `lock` varchar(128) NOT NULL DEFAULT '' COMMENT '锁',
 `updated` int(11) NOT NULL DEFAULT '0' COMMENT '最后的更新时间，用于keepalive',
 UNIQUE KEY `lock` (`lock`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8

CREATE TABLE `cursor` (
 `bin_file` varchar(128) NOT NULL DEFAULT '' COMMENT '最后读取到的binlog文件名',
 `pos` bigint(20) NOT NULL DEFAULT '0' COMMENT '读取到的位置',
 `event_index` bigint(20) NOT NULL DEFAULT '0' COMMENT '最新的事件索引',
 `updated` int(11) NOT NULL DEFAULT '0' COMMENT '最后更新的时间戳'
) ENGINE=InnoDB DEFAULT CHARSET=utf8
*/

// 校验当前节点是否为leader，返回true，说明是leader，则当前节点开始工作
// 返回false，说明是follower
func (m *Mysql) Leader() bool {
	if m.handler == nil {
		m.init()
	}
	if m.handler == nil {
		return false
	}
	sqlStr := "INSERT INTO `lock`(`ip`, `lock`, `updated`) VALUES (?,?,?)"
	log.Debugf("%s", sqlStr)
	ip := ""
	res, err := m.handler.Exec(sqlStr, ip, LOCK, time.Now().Unix())
	if err != nil {
		log.Errorf("insert db with error：%+v, %s, %d, %s", err, sqlStr, ip, LOCK)
		return false
	}
	num, err := res.RowsAffected()
	if err != nil {
		log.Errorf("insert db with error：%+v, %s, %d, %s", err, sqlStr, ip, LOCK)
		return false
	}
	if num > 0 {
		//如果获取锁成功
		m.lock.Lock()
		m.IsLeader = true
		m.lock.Unlock()
		return true
	}
	return false
}

// 同步游标相关信息
func (m *Mysql) Write(binFile string, pos int64, eventIndex int64) bool {
	if m.handler == nil {
		m.init()
	}
	if m.handler == nil {
		return false
	}
	sqlStr := "UPDATE `cursor` SET `bin_file`=?,`pos`=?,`event_index`=?,`updated`=? WHERE 1"
	res, err := m.handler.Exec(sqlStr, binFile, pos, eventIndex, time.Now().Unix())
	if err != nil {
		log.Errorf("query with error：%+v, %s, %s, %d, %d", err, sqlStr, binFile, pos, eventIndex)
		return false
	}
	num, err := res.RowsAffected()
	if err != nil {
		log.Errorf("query with error：%+v, %s, %s, %d, %d", err, sqlStr, binFile, pos, eventIndex)
		return false
	}
	return num > 0
}

// 读取游标信息
// 返回binFile string, pos int64, eventIndex int64
func (m *Mysql) Read() (string, int64, int64) {
	if m.handler == nil {
		m.init()
	}
	if m.handler == nil {
		return "", 0, 0
	}
	sqlStr := "select * from `cursor`"
	row := m.handler.QueryRow(sqlStr)
	var (
		binFile string
		pos int64
		eventIndex int64
	)
	err := row.Scan(&binFile, &pos, &eventIndex)
	if err != nil {
		log.Errorf("query with error：%+v, %s, %s, %d, %d", err, sqlStr, binFile, pos, eventIndex)
		return "", 0, 0
	}
	return binFile, pos, eventIndex
}

// 获取当前的集群成员
func (d *Mysql) Members() []ClusterMember {
	return nil
}

func (d *Mysql) AddMember() bool {
	return false
}

func (d *Mysql) DelMember() bool {
	return false
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
	//读取配置
	//连接mysql
	go d.keepAlive()
	go d.checkTimeout()
}

