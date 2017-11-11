#!/usr/bin/env bash
##杀掉正在运行的进程然后再运行，避免重复运行
kill -9 $(ps -ef|grep wing-binlog-go|gawk '$0 !~/grep/ {print $2}' |tr -s '\n' ' ')
./bin/wing-binlog-go 1>./bin/logs/wing-binlog-go.log 2>./bin/logs/wing-binlog-go.log 3>./bin/logs/wing-binlog-go.log &