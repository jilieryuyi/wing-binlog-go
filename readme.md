wing-binlog-go
----
This project uses replication protocol to read events from Mysql or MariaDB's binlog build on top of [go-mysql](https://github.com/siddontang/go-mysql/), and send json data via Http, TCP or WebSocket.
It allows you to receive events like insert, update and delete with the original data.

* wing-binlog-go is a high performance Mysql/MariaDB middleware
* wing-binlog-go is a light weight Mysql/MariaDB data watch system

wing-binlog-go can run on OSX, Linux and Windows system.

Installation:
* On Xnix, run sh build.sh
* On Windows, run build.bat

Project status
The project is test with:
* Mysql 5.5, 5.6 and 5.7
* MariaDB 5.5, 10.0, 10.1 and 10.2
* golang version 1.8+
* consul v1.0.2

Important Mysql server settings
```
[mysqld]
expire_logs_days         = 10
max_binlog_size          = 100M
server-id		 = 1
log_bin			 = /var/log/mysql/mysql-bin.log
binlog-format            = row
```
The system variables server_id, log_bin and binlog-format are madantory, and row based replication is required for the project.


Features:
* TCP protocol support
* WebSocket protocol support
* Http protocal support
* Support client group, broadcast, load balance and filters
* Auto remove bad http node,

Use cases:
* MySQL to NoSQL replication
* MySQL to search engine replication, such as elasticsearch
* Realtime analytics
* Audit

Usage:
* Help: ./wing-binlog-go -help
* Start service: ./wing-binlog-go
* Show version information: ./wing-binlog-go -version
* Stop service: ./wing-binlog-go -stop
* Reload service: ./wing-binlog-go -service-reload all|http|tcp

Known issues:
1. The underlying package go-mysql does not support compressed binlog
2. Schema changes after initial binlog position may cause incorrect json package

Special thanks:
* https://github.com/go-sql-driver/mysql
* https://github.com/larspensjo/config
* https://github.com/siddontang/go-mysql
* https://github.com/BurntSushi/toml
* https://github.com/go-martini/martini
* https://github.com/gorilla/websocket
* https://github.com/garyburd/redigo/redis
* https://github.com/takama/daemon
* https://github.com/mattn/go-sqlite3
* https://github.com/segmentio/kafka-go
* https://github.com/axgle/mahonia

Contributors
* jilieryuyi
* mia0x75


Help:

QQ group: 535218312
