#!/usr/bin/env bash
echo "building..."
current_path=$(cd `dirname $0`; pwd)
root_path=$(dirname $current_path)

bin_path="${root_path}/bin"
pkg_path="${root_path}/pkg"
vendor_path="${root_path}/vendor"

##杀掉正在运行的进程然后再编译
##kill -9 $(ps -ef|grep wing-binlog-go|gawk '$0 !~/grep/ {print $2}' |tr -s '\n' ' ')

##添加当前目录和当前目录下的vendor目录到GOPATH环境变量
export GOPATH="${root_path}/vendor:${root_path}"

##如果pkg目录存在，则删除
if [ -d "${pkg_path}" ]
then
	rm -rf "${pkg_path}"
fi

if [ ! -d "${vendor_path}" ]
then
	mkdir "${vendor_path}"
	mkdir "${vendor_path}/src"
	sh "${root_path}/bin/vendor_install.sh"
fi

##进入当前目录
cd ${root_path}
##build构建项目
go build -p 4 -race wing-binlog-go ##-a强制重新编译所有的包 -v显示被编译的包 -x显示所用到的其他命令

##编译不成功则退出
if [[ $? -ne 0 ]]
then
	echo "An error occurred during the compiling"
	exit $?
fi

##install安装
go install wing-binlog-go
##删除根目录下的可执行文件
rm wing-binlog-go

##配置文件目录不存在即复制配置文件
if [ ! -d "${bin_path}/config" ]
then
    mkdir ${root_path}/bin/config/
	cp -rf ${root_path}/src/config/* ${root_path}/bin/config/
fi

##Web文件目录不存在即复制Web文件
#if [ ! -d "${bin_path}/web" ]
#then
#	mkdir ${root_path}/bin/web/
#fi

##cp -rf ${root_path}/src/library ${root_path}/vendor/
##cp -rf ${vendor_path}/src/* ${vendor_path}
##cp -rf ${root_path}/web/* ${root_path}/bin/web/

echo "build success"
echo ${root_path}/bin/wing-binlog-go

