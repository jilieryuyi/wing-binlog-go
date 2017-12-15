<?php
/**
 * 运行方式 php -S 0.0.0.0:8000 http.php
 * 监听在本地 8000端口
 * 发送端可以配置将数据发送到http://127.0.0.1:8000/
 */

// 获取centineld通过POST发过来的数据
$content = file_get_contents('php://input', 'r');

// 直接在控制台输出
file_put_contents("php://stdout", "\n收到数据: ${content}");

// 输出信息给调用端，可省
echo 'HTTP 1.1 200 OK';
