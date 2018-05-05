#!/usr/bin/env bash
##启动集群
./kafka_2.12-1.1.0/bin/kafka-server-start.sh -daemon ./kafka_2.12-1.1.0/config/server.properties
./kafka_2.12-1.1.0/bin/kafka-server-start.sh -daemon ./kafka_2.12-1.1.0/config/server-1.properties
./kafka_2.12-1.1.0/bin/kafka-server-start.sh -daemon ./kafka_2.12-1.1.0/config/server-2.properties


