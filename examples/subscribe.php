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

$client = new \Wing\Binlog\Go\Client("127.0.0.1", 9996);
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
    //注意这里如果需要打印，务必使用fwrite STDERR的模式
    //否则有可能多次重建子进程以后，由于多进程输出重定向的原因，echo和var_dump是看不到的
    //\Wing\Binlog\Go\Client::debug api 默认输出到STDERR
    \Wing\Binlog\Go\Client::debug("收到新的事件" . json_encode($data) . "\r\n");
});
//开始接受数据
$client->start();
//父进程，等待子进程退出
$client->wait();
