基于mysql数据库binlog的分布式增量订阅&消费
====
>wing-binlog-go是一个高性能数据中间件    
wing-binlog是一个轻量化mysql数据库监控系统       
wing-binlog可轻松实现不改变一句业务代码实现整库数据变化实时监控          
......              

https://github.com/jilieryuyi/wing-binlog

mac, linux 或者其他 unix 系统，只需要运行 build.sh 即可完成全自动化构建         
windows下运行 build.bat 即可    
  
特点-Feature
----    
1、支持tcp服务协议 - 支持分组、广播、负载均衡、过滤器      
2、支持websocket服务协议 - 支持分组、广播、负载均衡、过滤器      
3、支持http服务协议- 支持分组、广播、负载均衡、过滤器      
4、支持http节点故障熔断移除、自动检测恢复与数据重发     
5、支持优雅的服务退出         

使用-Usage
----
1、./bin/wing-binlog-go 启动服务       
2、./bin/wing-binlog-go -version 查看版本  
3、./bin/wing-binlog-go -stop 停止服务  

已知问题-Known issues
----
1、DDL操作（数据表结构表更）      
在启动wing-binlog-go之前发生变化的数据，由于数据表结构发生变化，对过去的数据是不可见的，
所以，删除掉某个字段以后，过去的数据这个字段也会受影响（字段不存在），对于新增的字段，默认为NULL，另外字段和值的映射关系也不可保证。        
<b>如果wing-binlog-go运行时改变数据表结构，无此问题</b>     
2、暂时不支持log_bin_compress选项     

帮助-Help
----
QQ群：535218312