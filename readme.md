基于mysql数据库binlog的分布式增量订阅&消费
====
>此仓库为wing-binlog的go版本          
开发中...          

https://github.com/jilieryuyi/wing-binlog

mac, linux 或者其他 unix 系统，只需要运行 build.sh 即可完成全自动化构建         
windows下运行 build.bat 即可    
  
future
----    
1、高可用，支持分布式多节点部署    
2、去中心化    
3、多协议支持（tcp、websocket、http、redis、kafka、ssh）    
4、自带负载均衡    
5、节点以及订阅者支持容错和自动检测恢复

已知问题-Known issues
----
DDL操作（数据表结构表更）      
在启动wing-binlog-go之前发生变化的数据，由于数据表结构发生变化，对过去的数据是不可见的，
所以，删除掉某个字段以后，过去的数据这个字段也会受影响（字段不存在），对于新增的字段，默认为NULL，另外字段和值的映射关系也不可保证。    
      
<b>如果wing-binlog-go运行时改变数据表结构，无此问题</b>    
