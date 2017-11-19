#!/usr/bin/env bash
echo "building..."
current_path=$(cd `dirname $0`; pwd)
bin_path=$current_path"/bin"
pkg_path=$current_path"/pkg"
vendor_path=$current_path"/vendor"

##杀掉正在运行的进程然后再编译
##kill -9 $(ps -ef|grep wing-binlog-go|gawk '$0 !~/grep/ {print $2}' |tr -s '\n' ' ')

##添加当前目录和当前目录下的vendor目录到GOPATH环境变量
export GOPATH="$current_path/vendor:$current_path"

##如果bin目录存在，则删除
##if [ -d "$bin_path" ]; then
## rm -rf "$bin_path"
##fi

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
go build -p 4 -race wing-binlog-go ##-a强制重新编译所有的包 -v显示被编译的包 -x显示所用到的其他命令
##install安装
go install wing-binlog-go
##删除根目录下的可执行文件
rm wing-binlog-go

if [ ! -d "$bin_path/config" ]; then
mkdir "$bin_path/config"
fi
if [ ! -d "$bin_path/web" ]; then
mkdir "$bin_path/web"
fi

##拷贝配置文件
cp -rf $current_path/src/config/* $current_path/bin/config/
cp -rf $current_path/web/* $current_path/bin/web/

echo "build success"
echo $current_path/bin/wing-binlog-go