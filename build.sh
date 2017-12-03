#!/usr/bin/env bash
echo "building..."
current_path=$(cd `dirname $0`; pwd)
bin_path="${current_path}/bin"
pkg_path="${current_path}/pkg"
vendor_path="${current_path}/vendor"

##杀掉正在运行的进程然后再编译
##kill -9 $(ps -ef|grep wing-binlog-go|gawk '$0 !~/grep/ {print $2}' |tr -s '\n' ' ')

##添加当前目录和当前目录下的vendor目录到GOPATH环境变量
export GOPATH="${current_path}/vendor:${current_path}"

##如果pkg目录存在，则删除
if [ -d "${pkg_path}" ]
then
	rm -rf "${pkg_path}"
fi

if [ ! -d "${vendor_path}" ]
then
	mkdir "${vendor_path}"
	mkdir "${vendor_path}/src"
	sh "${current_path}/vendor_install.sh"
fi

##进入当前目录
cd ${current_path}
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
	cp -rf ${current_path}/src/config/ ${current_path}/bin/config/
fi

##Web文件目录不存在即复制Web文件
##if [ ! -d "${bin_path}/web" ]
##then
	cp -rf ${current_path}/web/ ${current_path}/bin/web/
##fi

##cp -rf ${current_path}/src/library ${current_path}/vendor/
##cp -rf ${vendor_path}/src/* ${vendor_path}

echo "build success"
echo ${current_path}/bin/wing-binlog-go

