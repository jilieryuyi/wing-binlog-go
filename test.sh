#!/usr/bin/env bash
current_path=$(cd `dirname $0`; pwd)
library_ip_path=$current_path"/src/library/ip"
library_debug_path=$current_path"/src/library/debug"
library_path_path=$current_path"/src/library/path"
library_std_path=$current_path"/src/library/std"
library_path=$current_path"/src/library"

export GOPATH="$current_path/vendor:$current_path"

cd $library_ip_path && go test
cd $library_debug_path && go test
cd $library_path_path && go test
cd $library_std_path && go test
cd $library_path && go test