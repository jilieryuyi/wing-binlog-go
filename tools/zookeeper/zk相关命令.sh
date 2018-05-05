mkdir zookeeper
cd zookeeper
mkdir data
mkdir logs

wget http://mirrors.cnnic.cn/apache/zookeeper/zookeeper-3.4.11/zookeeper-3.4.11.tar.gz
tar -zxvf zookeeper-3.4.11.tar.gz

cd zookeeper-3.4.11/conf
cp zoo_sample.cfg zoo.cfg

修改配置文件

#启动服务
./zkServer.sh start
#检查服务器状态
./zkServer.sh status
./zkServer.sh stop


ssh root1@10.10.109.16
密码 vRQKJ1j1z9QM

  1 # The number of milliseconds of each tick
  2 tickTime=2000
  3 # The number of ticks that the initial
  4 # synchronization phase can take
  5 initLimit=10
  6 # The number of ticks that can pass between
  7 # sending a request and getting an acknowledgement
  8 syncLimit=5
  9 # the directory where the snapshot is stored.
 10 # do not use /tmp for storage, /tmp here is just
 11 # example sakes.
dataDir=/usr/local/zookeeper/data
dataLogDir=/usr/local/zookeeper/logs
 14
 15 # the port at which the clients will connect
 16 clientPort=12181
 17 # the maximum number of client connections.
 18 # increase this if you need to handle more clients
 19 #maxClientCnxns=60
 20 #
 21 # Be sure to read the maintenance section of the
 22 # administrator guide before turning on autopurge.
 23 #
 24 # http://zookeeper.apache.org/doc/current/zookeeperAdmin.html#sc_maintenance
 25 #
 26 # The number of snapshots to retain in dataDir
 27 autopurge.snapRetainCount=3
 28 # Purge task interval in hours
 29 # Set to "0" to disable auto purge feature
 30 autopurge.purgeInterval=1

server.1=node1:12888:13888
server.2=node2:12888:13888
server.3=node3:12888:13888

10.10.131.131 node1
10.10.109.15 node2
10.10.109.16 node3

./zkServer.sh start-foreground

Error contacting service. It is probably not running.
问题解决

https://blog.csdn.net/sinat_32873711/article/details/78450431
yum install java 默认的安装路径如下
/usr/lib/jvm/java-1.8.0-openjdk-1.8.0.102-4.b14.el7.x86_64
配置环境变量
source /etc/profile


以上问题出现的原因，
myid 跟对应的host名称写错了



