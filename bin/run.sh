#!/usr/bin/env bash
##杀掉正在运行的进程然后再运行，避免重复运行
kill -9 $(ps -ef|grep wing-binlog-go|gawk '$0 !~/grep/ {print $2}' |tr -s '\n' ' ')
wing-binlog-go 1>./logs/wing-binlog-go.log 2>./logs/wing-binlog-go.log 3>./logs/wing-binlog-go.log &