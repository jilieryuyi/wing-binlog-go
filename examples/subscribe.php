<?php
/**
 * Created by PhpStorm.
 * User: yuyi
 * Date: 2018/4/17
 * Time: 17:45
 */
//引入核心php库
include "Client.php";
date_default_timezone_set('Asia/Shanghai');

$defaultServiceIp   = "127.0.0.1";
$defaultServicePort = 9996;

if ($argc > 1) {
    $temp = explode(":", $argv[1]);
    $defaultServiceIp   = $temp[0];
    $defaultServicePort = $temp[1];
}

$client = new \Wing\Binlog\Go\Client($defaultServiceIp, $defaultServicePort);
register_shutdown_function(function () use($client) {
    //如果wait返回了，说明所有的进程都退出了
    //这里需要对socket进行关闭处理
    $client->close();
});

//订阅感兴趣的数据变化
//这里的订阅参数是 database.table
//支持正则和订阅多个主题
$client
    ->subscribe("new_yonglibao_c.*")
    ->subscribe("test.*");

// 注册事件回调
$client->setOnEvent(function ($data) {
    /**
     * $data 是一个数组，字段如下
     * database 发生变化的数据库
     * event 事件数据
     * data 为具体变化的数据
     *      内部字段对应数据库的字段
     * --data字段，仅update的时候分为old_data和new_data，其他的事件数据一致
     * event_index 唯一事件id
     * event_type 事件类型，目前所有的事件类型为 update、insert、delete、alter
     * time 事件发生时间

     alter事件是数据表结构变化事件，如：
     * {"database":"test","event_index":1164,"event_type":"alter","table":"bar","time":1524116248}

    delete 事件数据格式
    {
    "database": "new_yonglibao_c",
    "event": {
        "data": {
            "city_name": "梧州市",
            "id": 5764808,
            "provinces_id": 22
        }
    },
    "event_index": 1161,
    "event_type": "delete",
    "table": "bl_city1",
    "time": 1524115287
    }
     *
     update
    {
    "database": "new_yonglibao_c",
    "event": {
        "data": {
            "new_data": {
                "city_name": "北海市1",
                "id": 5764809,
                "provinces_id": 22
            },
            "old_data": {
                "city_name": "北海市",
                "id": 5764809,
                "provinces_id": 22
            }
    }
    },
    "event_index": 1162,
    "event_type": "update",
    "table": "bl_city1",
    "time": 1524115867
    }
     *
     insert
    {
    "database": "new_yonglibao_c",
    "event": {
    "data": {
    "city_name": "哈哈哈",
    "id": 6078191,
    "provinces_id": 1
    }
    },
    "event_index": 1163,
    "event_type": "insert",
    "table": "bl_city1",
    "time": 1524115914
    }
     */
    //注意这里如果需要打印，务必使用fwrite STDERR的模式
    //否则有可能多次重建子进程以后，由于多进程输出重定向的原因，echo和var_dump是看不到的
    //\Wing\Binlog\Go\Client::debug api 默认输出到STDERR
    \Wing\Binlog\Go\Client::debug("收到新的事件" . json_encode($data));
});
//开始接受数据
$client->start();
//父进程，等待子进程退出
$client->wait();
