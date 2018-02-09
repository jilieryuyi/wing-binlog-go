<?php
/**
 * Created by PhpStorm.
 * User: yuyi
 * Date: 17/11/12
 * Time: 07:51
 */

const CMD_SET_PRO = 1;
const CMD_AUTH    = 2;
const CMD_OK      = 3;
const CMD_ERROR   = 4;
const CMD_TICK    = 5;
const CMD_EVENT   = 6;

const MODE_BROADCAST = 1; //广播
const MODE_WEIGHT    = 2; //权重

date_default_timezone_set('Asia/Shanghai');

// 打印debug信息，直接输出的标准错误，这样可以禁止输出缓存
function clog($content)
{
    fwrite(STDERR, date("Y-m-d H:i:s") . " " . $content . "\r\n");
}

// 信号处理
function sig_handler()
{
    clog("收到退出信号");
    exit;
}

// 数据打包
function pack_cmd($cmd, $content = "")
{
	$l = strlen($content) + 2;
	$r = "";

	// 4字节数据包长度
	$r .= chr($l);
	$r .= chr($l >> 8);
    $r .= chr($l >> 16);
    $r .= chr($l >> 32);

    // 2字节cmd
	$r .= chr($cmd);
	$r .= chr($cmd >> 8);

	// 实际美容
	$r .= $content;

	return $r;
}

// 创建子进程，用于发送心跳包
function fork_child($socket)
{
    $pid = pcntl_fork();
    if ($pid > 0) {
        return $pid;
    }

    pcntl_signal(SIGINT,  "sig_handler", false);

    $tick = pack_cmd(CMD_TICK);
    //子进程发送心跳包
    while(1) {
        pcntl_signal_dispatch();
        try {
            clog("发送心跳包");
            socket_write($socket, $tick);
            // 3秒发送一次
            sleep(3);
        } catch (\Exception $e) {
            clog($e->getMessage());
            exit;
        }
    }
    return $pid;
}

// 开始服务
function start_service()
{
    pcntl_signal(SIGINT,  "sig_handler", false);

    $socket = socket_create(AF_INET, SOCK_STREAM, SOL_TCP);
    $con    = socket_connect($socket, '127.0.0.1', 9998);

    if (!$con) {
        socket_close($socket);
        clog("无法连接服务器，等待重试");
        return false;
    }

    //注册到分组
    //3秒之内不发送加入到分组将被强制断开
    $pack = pack_cmd(CMD_SET_PRO, "group1");
    socket_write($socket, $pack);

    //测试
    //echo "连接关闭\r\n";
    //socket_shutdown($socket);
    //socket_close($socket);
    //return;

    clog("连接成功");
    $child = fork_child($socket);

    //父进程接收消息
    $count    = 1;
    $recv_buf = "";
    $start_time = time();

    while ($msg = socket_read($socket, 4096)) {
        ob_start();
        pcntl_signal_dispatch();

        $recv_buf .= $msg;

        while (1) {
            //循环处理所有的缓冲数据
            if (strlen($recv_buf) <= 0) {
                break;
            }

            // 包长度4字节
            $len = (ord($recv_buf[0])) + (ord($recv_buf[1]) << 8) +
                (ord($recv_buf[2]) << 16) + (ord($recv_buf[3]) << 32);

            // 接收到的包还不完整，继续等待
            if (strlen($recv_buf) < $len + 4) {
                break;
            }

            // cmd长度2字节
            $cmd      = (ord($recv_buf[4])) + (ord($recv_buf[5]) << 8);
            // 开始的4字节是长度，接下来的2字节是cmd，所有内容从6开始，长度为 $len - 2字节的cmd长度
            $content  = substr($recv_buf, 6, $len - 2);
            // 删除掉已经读取的数据
            $recv_buf = substr($recv_buf, $len + 4);

            $s = time() - $start_time;
            $p = 0;
            if ($s > 0) {
                $p = $count/$s;
            }
            switch ($cmd) {
                case CMD_TICK:
                    clog("心跳包返回值：" . $content);
                    break;
                case CMD_ERROR:
                    clog("错误：" . $content);
                    break;
                case CMD_EVENT:
                    clog("每秒响应 ".$p." 次，".$count . "次收到事件：" . $content);
                    $count++;
                    break;
                case CMD_OK:
                    //心跳包会回复这个消息
                    clog("成功响应：" . $content);
                    break;
                case CMD_SET_PRO:
                    //发送注册到分组，响应的cmd也会是注册到分组
                    clog("注册到分组操作成功响应：" . $content);
                    break;
                default:
                    clog("未知事件：" . $cmd);
            }
        }
        $content = ob_get_contents();
        ob_end_clean();
        echo $content;
    }

    clog("连接关闭");
    socket_shutdown($socket);
    socket_close($socket);

    //简单的子进程管理，当父进程退出时
    $start = 0;
    while (1) {
        $status = 0;
        pcntl_signal_dispatch();
        posix_kill($child, SIGINT);
        $pid = pcntl_wait($status);
        if ($pid > 0) {
            break;
        }
        if ((time() - $start) > 5) break;
    }
    return true;
}

//打开while 1，断开将自动重连
while (1) {
    if (false === start_service()) {
        sleep(1);
    }
}
