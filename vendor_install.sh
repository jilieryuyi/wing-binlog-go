#!/usr/bin/env bash
current_path=$(cd `dirname $0`; pwd)
vendor_path=$current_path"/vendor"

##添加当前目录和当前目录下的vendor目录到GOPATH环境变量
export GOPATH="$vendor_path:$current_path"
if [ ! -d "$vendor_path" ]; then
 mkdir "$vendor_path"
 mkdir "$vendor_path/src"
fi
echo "installing... go-sql-driver/mysql"
go get github.com/go-sql-driver/mysql
echo "installing... larspensjo/config"
go get github.com/larspensjo/config
echo "installing... siddontang/go-mysql/canal"
go get github.com/siddontang/go-mysql/canal
echo "installing... siddontang/go-mysql/replication"
go get github.com/siddontang/go-mysql/replication
echo "installing... siddontang/go-mysql/mysql"
go get github.com/siddontang/go-mysql/mysql
echo "installing... BurntSushi/toml"
go get github.com/BurntSushi/toml
echo "installing... go-martini/martini"
go get github.com/go-martini/martini
echo "installing... gorilla/websocket"
go get github.com/gorilla/websocket
echo "installing... garyburd/redigo/redis"
go get github.com/garyburd/redigo/redis

find $vendor_path -name '*.git*' | xargs rm -rf
cp -rf $vendor_path/src/* $vendor_path
echo "install complete"
