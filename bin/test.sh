#!/usr/bin/env bash

function getDir()
{
    for element in `ls $1`
    do
        dir_or_file=$1"/"$element
        if [ -d $dir_or_file ]
        then
            cd $dir_or_file && go test
            getDir $dir_or_file
        fi
    done
}

current_path=$(cd `dirname $0`; pwd)
root_path=$(dirname $current_path)
library_ip_path=$root_path"/src/library/ip"
library_debug_path=$root_path"/src/library/debug"
library_path=$root_path"/src/library"
ini_test_file=$root_path"/src/config/mysql.toml"

export GOPATH="$root_path/vendor:$root_path"
##用于文件测试
##echo 123 >/tmp/__test.txt
##用户配置文件测试
##cp $ini_test_file "/tmp/__test_mysql.toml"

root_dir="$root_path/src"
getDir $root_dir

##cd $library_ip_path && go test
##cd $library_debug_path && go test
##cd $library_path && go test
##删除临时的测试文件
#rm /tmp/__test_mysql.toml
#rm /tmp/__test.txt