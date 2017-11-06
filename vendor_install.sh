#!/usr/bin/env bash
current_path=$(cd `dirname $0`; pwd)
vendor_path=$current_path"/vendor"

##添加当前目录和当前目录下的vendor目录到GOPATH环境变量
export GOPATH="$current_path/vendor:$current_path"
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

cp -rf $vendor_path/src/* $vendor_path


function rmGit()
{
    for element in `ls $1`
    do
        dir_or_file=$1"/"$element
        if [ -d $dir_or_file ]
        then
            rm -rf $dir_or_file"/.git"
            rmGit $dir_or_file
        fi
    done
}

rmGit $vendor_path