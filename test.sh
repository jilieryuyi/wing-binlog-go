#!/usr/bin/env bash
current_path=$(cd `dirname $0`; pwd)
library_ip_path=$current_path"/src/library/ip"
library_debug_path=$current_path"/src/library/debug"
library_path=$current_path"/src/library"
ini_test_file=$current_path"/src/config/mysql.ini"

export GOPATH="$current_path/vendor:$current_path"
echo 123 > /tmp/__test.txt
cp $ini_test_file "/tmp/__test_mysql.ini"


function getDir() {
    for element in `ls $1`
    do
        dir_or_file=$1"/"$element
        if [ -d $dir_or_file ]
        then
            cd $dir_or_file && go test
            getDir $dir_or_file
        ##else
            ##echo $dir_or_file
        fi
    done
}

root_dir="$current_path/src"
getDir $root_dir

##cd $library_ip_path && go test
##cd $library_debug_path && go test
##cd $library_path && go test
rm /tmp/__test_mysql.ini
rm /tmp/__test.txt