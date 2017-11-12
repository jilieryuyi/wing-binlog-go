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

function clog($content)
{
    fwrite(STDERR, date("Y-m-d H:i:s") . " " . $content . "\r\n");
}

// 数据打包
function pack_cmd($cmd, $content = "")
{
	$l = strlen($content) + 2;
	$r = "";

	$r .= chr($l);
	$r .= chr($l >> 8);
    $r .= chr($l >> 16);
    $r .= chr($l >> 32);

	$r .= chr($cmd);
	$r .= chr($cmd >> 8);
	$r .= $content;

	return $r;
}

function pack_set_pro($group_name, $weight)
{
    $l = 6 + strlen($group_name);
    $r = "";

    $r .= chr($l);
    $r .= chr($l >> 8);
    $r .= chr($l >> 16);
    $r .= chr($l >> 32);

    $r .= chr(CMD_SET_PRO);
    $r .= chr(CMD_SET_PRO >> 8);

    $r .= chr($weight);
    $r .= chr($weight >> 8);
    $r .= chr($weight >> 16);
    $r .= chr($weight >> 32);

    $r .= $group_name;

    return $r;
}

// 创建子进程，用于发送心跳包
function fork_child($socket)
{
    $pid = pcntl_fork();
    if ($pid > 0) {
        return;
    }

    $tick = pack_cmd(CMD_TICK);
    //子进程发送心跳包
    while(1) {
        try {
            socket_write($socket, $tick);
            // 3秒发送一次
            sleep(3);
        } catch (\Exception $e) {
            clog($e->getMessage());
            exit;
        }
    }
}


function start_service()
{
    $socket = socket_create(AF_INET, SOCK_STREAM, SOL_TCP);
    $con    = socket_connect($socket, '127.0.0.1', 9998);

    if (!$con) {
        socket_close($socket);
        clog("无法连接服务器");
        exit;
    }

    //注册到分组
    //两种模式，广播和权重，决定以那种模式注册到服务端是分组决定的
    //wing-binclog-go/src/config/tcp.toml如配置文件下的group1模式为广播
    //如果以广播的方式注册到服务端，那每次都会收到事件
    //如果以权重的方式注册到服务端，服务端会按照权重比例，均衡的把事件发送给已经注册到服务端的客户端
    //权重值 0 - 100，当然也只有分组模式为 MODE_WEIGHT 时有效，为广播时此值会被忽略
    //最后一个值得意思是要注册到那个分组
    $pack = pack_set_pro("group1", 100);
    socket_write($socket, $pack);

    clog("连接成功");
    fork_child($socket);

    //父进程接收消息
    $count    = 0;
    $recv_buf = "";

    while ($msg = socket_read($socket, 4096)) {
        $recv_buf .= $msg;

        while (1) {
            //循环处理所有的缓冲数据
            if (strlen($recv_buf) <= 0) {
                break;
            }

            $len = (ord($recv_buf[0])) + (ord($recv_buf[1]) << 8) +
                (ord($recv_buf[2]) << 16) + (ord($recv_buf[3]) << 32);

            // 接收到的包好不完整，继续等待
            if (strlen($recv_buf) < $len + 4) {
                break;
            }

            $cmd      = (ord($recv_buf[4])) + (ord($recv_buf[5]) << 8);
            // 开始的4字节是长度，接下来的2字节是cmd，所有内容从6开始，长度为 $len - 2字节的cmd长度
            $content  = substr($recv_buf, 6, $len - 2);

            // 删除掉已经读取的数据
            $recv_buf = substr($recv_buf, $len + 4);

            switch ($cmd) {
                case CMD_TICK:
                    clog("收到心跳包：" . $content);
                    break;
                case CMD_ERROR:
                    clog("错误：" . $content);
                    break;
                case CMD_EVENT:
                    clog("收到事件：" . $content);
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
    }

    echo "连接关闭\r\n";
    socket_shutdown($socket);
    socket_close($socket);

    $start = 0;
    while (1) {
        $status = 0;
        $pid = pcntl_wait($status);
        if ($pid > 0) {
            break;
        }
        if ((time() - $start) > 5) break;
    }
}

while (1) {
    start_service();
}