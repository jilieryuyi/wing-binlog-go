package cluster

import (
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
	"database/sql"
	"fmt"
	"os"
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
	session string
}

const (
	LOCK = "wing-binlog-cluster-lock"
	TIMEOUT = 6 //6秒超时
)

func NewMysql() *Mysql {
	config, err := GetConfig()
	if err != nil {
		log.Printf("cluster mysql with error: %+v", err)
	}
	m := &Mysql{
		addr: config.Mysql.Addr,
		port: config.Mysql.Port,
		user: config.Mysql.User,
		password: config.Mysql.Password,
		database: config.Mysql.Database,
		charset: config.Mysql.Charset,
		lock: new(sync.Mutex),
		session: GetSession(),
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

// 校验当前节点是否为leader，返回true，说明是leader，则当前节点开始工作
// 返回false，说明是follower
func (m *Mysql) Leader() bool {
	if m.handler == nil {
		m.init()
	}
	if m.handler == nil {
		return false
	}
	sqlStr := "INSERT INTO `lock`(`session`, `lock`, `updated`) VALUES (?,?,?)"
	log.Debugf("%s", sqlStr)
	res, err := m.handler.Exec(sqlStr, m.session, LOCK, time.Now().Unix())
	if err != nil {
		log.Errorf("insert db with error：%+v, %s, %s", err, sqlStr, LOCK)
		return false
	}
	num, err := res.RowsAffected()
	if err != nil {
		log.Errorf("insert db with error：%+v, %s, %s", err, sqlStr, LOCK)
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
func (m *Mysql) updateLeader() bool {
	if m.handler == nil {
		m.init()
	}
	if m.handler == nil {
		return false
	}
	sqlStr := "UPDATE `lock` SET `updated`=? WHERE `session`=?"
	res, err := m.handler.Exec(sqlStr, time.Now().Unix(), m.session)
	if err != nil {
		log.Errorf("insert db with error：%+v, %s", err, sqlStr)
		return false
	}
	num, err := res.RowsAffected()
	if err != nil {
		log.Errorf("insert db with error：%+v, %s", err, sqlStr)
		return false
	}
	return num > 0
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
func (d *Mysql) Members() []*ClusterMember {
	return nil
}

func (m *Mysql) GetMember() *ClusterMember {
	if m.handler == nil {
		m.init()
	}
	if m.handler == nil {
		return nil
	}
	sqlStr := "SELECT * FROM `members` WHERE `session`=?"
	row := m.handler.QueryRow(sqlStr)
	var (
		id int
		hostname string
		session string
		isLeader int
		updated int64
	)
	err := row.Scan(&id, &hostname, &session, &isLeader, &updated)
	if err != nil {
		log.Errorf("query with error：%+v, %s", err, sqlStr)
		return nil
	}
	return &ClusterMember{
		Hostname:hostname,
		IsLeader:isLeader == 1,
		Updated:updated,
	}
}

func (m *Mysql) AddMember(isLeader bool) bool {
	if m.handler == nil {
		m.init()
	}
	if m.handler == nil {
		return false
	}
	sqlStr := "INSERT INTO `members`(`hostname`, `session`, `is_leader`, `updated`) VALUES (?,?,?,?)"
	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
		log.Errorf("get hostname with error: %+v", err)
	}
	iIsLeader := 0
	if isLeader {
		iIsLeader = 1
	}
	res, err := m.handler.Exec(sqlStr, hostname, m.session, iIsLeader, time.Now().Unix())
	if err != nil {
		log.Errorf("insert db with error：%+v, %s", err, sqlStr)
		return false
	}
	num, err := res.RowsAffected()
	if err != nil {
		log.Errorf("insert db with error：%+v, %s", err, sqlStr)
		return false
	}
	return num > 0
}

func (m *Mysql) UpdateMember(isLeader bool) bool {
	if m.handler == nil {
		m.init()
	}
	if m.handler == nil {
		return false
	}
	sqlStr := "UPDATE `members` SET `hostname`=?,`is_leader`=?,`updated`=? WHERE `session`=?"
	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
		log.Errorf("get hostname with error: %+v", err)
	}
	iIsLeader := 0
	if isLeader {
		iIsLeader = 1
	}
	res, err := m.handler.Exec(sqlStr, hostname, iIsLeader, time.Now().Unix(), m.session)
	if err != nil {
		log.Errorf("insert db with error：%+v, %s", err, sqlStr)
		return false
	}
	num, err := res.RowsAffected()
	if err != nil {
		log.Errorf("insert db with error：%+v, %s", err, sqlStr)
		return false
	}
	return num > 0
}

func (m *Mysql) DelMember() bool {
	if m.handler == nil {
		m.init()
	}
	if m.handler == nil {
		return false
	}
	sqlStr := "DELETE FROM `members` WHERE `session`=?"
	res, err := m.handler.Exec(sqlStr, m.session)
	if err != nil {
		log.Errorf("insert db with error：%+v, %s", err, sqlStr)
		return false
	}
	num, err := res.RowsAffected()
	if err != nil {
		log.Errorf("insert db with error：%+v, %s", err, sqlStr)
		return false
	}
	return num > 0
}

func (m *Mysql) keepAlive() {
	for {
		m.lock.Lock()
		if m.IsLeader {
			//leader节点保持心跳
			//更新updated为当前时间戳
			if m.updateLeader() {
				log.Debug("keep alive success")
			} else {
				log.Warn("keep alive success")
			}
		}
		m.lock.Unlock()
		time.Sleep(time.Second * 2)
	}
}

func (m *Mysql) checkTimeout() {
	for {
		m.lock.Lock()
		if !m.IsLeader {
			//非leader节点判断超时
			//读取updated字段与当前时间比对
			//如果超过TIMEOUT，ze重新选leader
		}
		m.lock.Unlock()
		time.Sleep(time.Second * 2)
	}
}

func (m *Mysql) Start() {
	//读取配置
	//连接mysql
	go m.keepAlive()
	go m.checkTimeout()
	isLeader := m.Leader()
	if nil == m.GetMember() {
		m.AddMember(isLeader)
	} else {
		m.UpdateMember(isLeader)
	}
}

