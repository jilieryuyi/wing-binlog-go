#!/usr/bin/env bash
current_path=$(cd `dirname $0`; pwd)
vendor_path=$current_path"/vendor/src"

mysql_path=$vendor_path"/mysql"
config_path=$vendor_path"/config"

if [ -d "$mysql_path" ]; then
 rm -rf "$mysql_path"
fi

if [ ! -d "$mysql_path" ]; then
 mkdir "$mysql_path"
fi

if [ -d "$config_path" ]; then
 rm -rf "$config_path"
fi

if [ ! -d "$config_path" ]; then
 mkdir "$config_path"
fi

git clone https://github.com/go-sql-driver/mysql.git $mysql_path
rm -rf $mysql_path"/.git"

git clone https://github.com/larspensjo/config.git $config_path
rm -rf $config_path"/.git"