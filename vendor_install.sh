#!/usr/bin/env bash
current_path=$(cd `dirname $0`; pwd)
vendor_path=$current_path"/vendor"

##添加当前目录和当前目录下的vendor目录到GOPATH环境变量
export GOPATH="$current_path/vendor:$current_path:$GOPATH"
if [ ! -d "$vendor_path" ]; then
 mkdir "$vendor_path"
 mkdir "$vendor_path/src"
fi

go get github.com/go-sql-driver/mysql
go get github.com/larspensjo/config
go get github.com/siddontang/go-mysql/canal
go get github.com/siddontang/go-mysql/replication
go get github.com/siddontang/go-mysql/mysql
go get github.com/BurntSushi/toml
#go get golang.org/x/crypto/ssh/terminal
#https://go.googlesource.com/crypto

cp -rf $vendor_path/src/* $vendor_path