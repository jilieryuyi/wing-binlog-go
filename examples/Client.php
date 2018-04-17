<?php
namespace Wing\Binlog\Go;
/**
 * Created by PhpStorm.
 * User: yuyi
 * Date: 17/11/12
 * Time: 07:51
 * only for linux
 */

class Client
{
    //php语言的客户端库
    const CMD_SET_PRO = 0;
    const CMD_AUTH = 1;
    const CMD_ERROR = 2;
    const CMD_TICK = 3;
    const CMD_EVENT = 4;

    private $ip;
    private $port;
    private $socket;
    private static $processes = [];
    private $onevent = [];
    private $keepalive_pid = 0;

    public function __construct($ip, $port)
    {
        $this->ip = $ip;
        $this->port = $port;

        $this->socket = \socket_create(AF_INET, SOCK_STREAM, SOL_TCP);
        $con = \socket_connect($this->socket, $this->ip, $this->port);
        self::debug("连接服务器" . $this->ip . ":" . $this->port);
        if (!$con) {
            \socket_close($this->socket);
            self::debug("无法连接服务器，等待重试");
        }
    }

    public function setOnEvent($f)
    {
        $this->onevent[] = $f;
    }

    // 打印debug信息，直接输出的标准错误，这样可以禁止输出缓存
    private static function debug($content)
    {
        fwrite(STDERR, date("Y-m-d H:i:s") . " " . $content . "\r\n");
    }

    // 信号处理
    public static function sig_handler()
    {
        self::debug("收到退出信号");
        self::stop();
        exit;
    }

    // 通用数据打包
    private static function pack_cmd($cmd, $content = "")
    {
        $l = strlen($content) + 2;
        $r = "";
        // 4字节数据包长度
        $r .= chr($l);
        $r .= chr($l >> 8);
        $r .= chr($l >> 16);
        $r .= chr($l >> 24);
        // 2字节cmd
        $r .= chr($cmd);
        $r .= chr($cmd >> 8);
        $r .= $content;
        return $r;
    }

    // 打包订阅分组的消息
    private static function pack_pro($content)
    {
        $l = strlen($content) + 3;
        $r = "";
        // 4字节数据包长度
        $r .= chr($l);
        $r .= chr($l >> 8);
        $r .= chr($l >> 16);
        $r .= chr($l >> 24);
        // 2字节cmd
        $r .= chr(self::CMD_SET_PRO);
        $r .= chr(self::CMD_SET_PRO >> 8);
        $r .= chr(0);
        $r .= $content;
        return $r;
    }

    // 心跳维持
    // 创建子进程，用于发送心跳包
    private static function keepalive($socket)
    {
        $pid = pcntl_fork();
        if ($pid > 0) {
            return $pid;
        }
        //\pcntl_signal(SIGINT, __CLASS__ . "::sig_handler", false);
        $tick = self::pack_cmd(self::CMD_TICK);
        //子进程发送心跳包
        while (1) {
            pcntl_signal_dispatch();
            try {
                //debug("发送心跳包");
                socket_write($socket, $tick);
                // 3秒发送一次
                sleep(3);
            } catch (\Exception $e) {
                self::debug($e->getMessage());
                exit;
            }
        }
        return $pid;
    }

    // 开始服务
    public function start()
    {
        self::debug("连接成功");
        self::$processes[] = $this->keepalive_pid = self::keepalive($this->socket);
        self::$processes[] = $readid = $this->read();
        self::debug("keepalive 进程".$this->keepalive_pid);
        self::debug("read 进程".$readid);
        return true;
    }

    // 开始读取消息
    // 这里创建了一个新的进程
    private function read()
    {
        $pid = pcntl_fork();
        if ($pid > 0) {
            return $pid;
        }
        //\pcntl_signal(SIGINT, __CLASS__ . "::sig_handler", false);
        //父进程接收消息
        //$count = 1;
        $recv_buf = "";
        //$start_time = time();

        while ($msg = socket_read($this->socket, 4096)) {
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
                    (ord($recv_buf[2]) << 16) + (ord($recv_buf[3]) << 24);

                // 接收到的包还不完整，继续等待
                if (strlen($recv_buf) < $len + 4) {
                    break;
                }

                // cmd长度2字节
                $cmd = (ord($recv_buf[4])) + (ord($recv_buf[5]) << 8);
                // 开始的4字节是长度，接下来的2字节是cmd，所有内容从6开始，长度为 $len - 2字节的cmd长度
                $content = substr($recv_buf, 6, $len - 2);
                // 删除掉已经读取的数据
                $recv_buf = substr($recv_buf, $len + 4);

//                $s = time() - $start_time;
//                $p = 0;
//                if ($s > 0) {
//                    $p = $count / $s;
//                }
                switch ($cmd) {
                    case self::CMD_TICK:
                        // debug("心跳包返回值：" . $content);
                        break;
                    case self::CMD_ERROR:
                        self::debug("错误：" . $content);
                        break;
                    case self::CMD_EVENT:
//                        self::debug("每秒响应 " . $p . " 次，" . $count . "次收到事件：" . $content);
//                        $count++;
                        $edata = json_decode($content, true);
                        foreach ($this->onevent as $f) {
                            $f($edata);
                        }
                        break;
                    case self::CMD_SET_PRO:
                        //发送注册到分组，响应的cmd也会是注册到分组
                        self::debug("订阅主题响应：" . $content);
                        break;
                    default:
                        self::debug("未知事件：" . $cmd);
                }
            }
            $content = ob_get_contents();
            ob_end_clean();
            echo $content;
        }

        self::debug("连接关闭");
        socket_shutdown($this->socket);
        socket_close($this->socket);
        return $pid;
    }

    // 关闭socket
    public function close()
    {
        socket_shutdown($this->socket);
        socket_close($this->socket);
    }

    // 订阅主题
    // 如果不订阅主题，默认对所有的事件变化感兴趣
    public function subscribe($topic)
    {
        $pack = self::pack_pro($topic);
        self::debug("发送注册分组");
        socket_write($this->socket, $pack);
        return $this;
    }

    // 退出服务
    public static function stop()
    {
        //简单的子进程管理，当父进程退出时
        $start = time();
        while (1) {
            $status = 0;
            \pcntl_signal_dispatch();
            foreach (self::$processes as $id => $child) {
                \posix_kill($child, SIGINT);
                $pid = \pcntl_wait($status);
               // $id = array_search($pid, self::$processes);
                if ($pid > 0) {
                    unset(self::$processes[$id]);
                    //var_dump(self::$processes);
                    self::debug($pid."退出成功");
                }
            }
            if (count(self::$processes) <= 0) {
                self::debug("退出成功");
                exit;
            }
            if ((time() - $start) > 5) {
                self::debug("退出超时");
//                var_dump(self::$processes);
//                foreach (self::$processes as $p) {
//                    exec("kill -9 ".$p);
//                }
                exit;
            }
        }
    }

    // 等待子进程退出
    public function wait()
    {
        self::debug("master进程id：".posix_getpid());
        \pcntl_signal(SIGINT, __CLASS__ . "::sig_handler", true);
        while (1) {
            \pcntl_signal_dispatch();
            try {
                \ob_start();
                $status = 0;
                $pid = \pcntl_wait($status, WNOHANG);
                if ($pid > 0) {
                    self::debug($pid . "进程退出");
                    $id = \array_search($pid, self::$processes);
                    unset(self::$processes[$id]);
                    if ($pid == $this->keepalive_pid) {
                        self::$processes[] = $this->keepalive_pid = self::keepalive($this->socket);
                        self::debug("keepalive进程退出，尝试重建，新进程id：".$this->keepalive_pid);
                    } else {
                        self::$processes[] = $readid = $this->read();
                        self::debug("read进程退出，尝试重建，新进程id：".$readid);
                    }
//                    \pcntl_signal(SIGINT, SIG_IGN);
//                    \pcntl_signal(SIGINT, __CLASS__ . "::sig_handler", false);
                }
                $content = \ob_get_contents();
                \ob_end_clean();
                if ($content) {
                    self::debug($content);
                }
            } catch (\Exception $e) {
                \var_dump($e->getMessage());
            }
            \sleep(1);
        }
    }
}