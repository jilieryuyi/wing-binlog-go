#!/usr/bin/env bash
current_path=$(cd `dirname $0`; pwd)
bin_path=$current_path"/bin"
pkg_path=$current_path"/pkg"
vendor_path=$current_path"/vendor"

##添加当前目录和当前目录下的vendor目录到GOPATH环境变量
export GOPATH="$current_path/vendor:$current_path"

##如果bin目录存在，则删除
if [ -d "$bin_path" ]; then
 rm -rf "$bin_path"
fi

##如果pkg目录存在，则删除
if [ -d "$pkg_path" ]; then
 rm -rf "$pkg_path"
fi

if [ ! -d "$vendor_path" ]; then
 mkdir "$vendor_path"
 mkdir "$vendor_path/src"
 sh $current_path"/vendor_install.sh"
fi

##进入当前目录
cd $current_path
##build构建项目
go build wing-binlog-go
##install安装
go install wing-binlog-go
##删除根目录下的可执行文件
rm wing-binlog-go

mkdir "$bin_path/config"

##拷贝配置文件
cp -rf "$current_path/src/config/" "$current_path/bin/config/"
echo "build success"