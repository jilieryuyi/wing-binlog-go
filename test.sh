#!/usr/bin/env bash
current_path=$(cd `dirname $0`; pwd)
library_ip_path=$current_path"/src/library/ip"
library_debug_path=$current_path"/src/library/debug"
library_path=$current_path"/src/library"
ini_test_file=$current_path"/src/config/mysql.ini"

export GOPATH="$current_path/vendor:$current_path"
cp $ini_test_file "/tmp"
cd $library_ip_path && go test
cd $library_debug_path && go test
cd $library_path && go test