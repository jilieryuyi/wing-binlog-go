wget http://mirrors.tuna.tsinghua.edu.cn/apache/kafka/1.1.0/kafka_2.12-1.1.0.tgz


./kafka-server-start.sh -daemon ../config/server.properties
./kafka-server-start.sh ../config/server.properties
#创建Topic
./kafka-topics.sh --create --zookeeper 127.0.0.1:12181 --replication-factor 1 --partitions 1 --topic shuaige

#创建一个broker，发布者
./kafka-console-producer.sh --broker-list 127.0.0.1:9092 --topic shuaige
#在一台服务器上创建一个订阅者
##./kafka-console-consumer.sh --bootstrap-server localhost:12181 --topic shuaige --from-beginning
./kafka-console-consumer.sh --zookeeper localhost:12181 --topic shuaige --from-beginning

##状态
kafka-topics.sh --describe --zookeeper localhost:12181 --topic shuaige
./kafka-topics.sh --describe --zookeeper localhost:12181 --topic testtopic

./kafka-console-consumer.sh --zookeeper localhost:12181 --topic wing-binlog-event --from-beginning

./kafka-server-stop.sh

./kafka-topics.sh --describe --zookeeper 127.0.0.1:12181 --topic wing-binlog-event

kafka-topics.sh --create --zookeeper 127.0.0.1:12181 --replication-factor 3 --partitions 1 --topic mytopic
kafka-topics.sh --describe --zookeeper 127.0.0.1:12181 --topic mytopic

./kafka-console-producer.sh --broker-list localhost:9092 --topic mytopic

./kafka-topics.sh --list --zookeeper localhost:12181
./kafka-topics.sh --zookeeper localhost:12181 --delete --topic testtopic4

replication-factor
./kafka-topics.sh --alter --zookeeper 127.0.0.1:12181 --replication-factor 3 --partitions 3 --topic wing-binlog-event
./kafka-topics.sh --zookeeper localhost:12181 --delete --topic wing-binlog-event




