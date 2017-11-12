var count = 0;
var group_name = "group1";
var weight = 100;

const CMD_SET_PRO = 1;
const CMD_AUTH    = 2;
const CMD_OK      = 3;
const CMD_ERROR   = 4;
const CMD_TICK    = 5;
const CMD_EVENT   = 6;

const MODE_BROADCAST = 1; //广播
const MODE_WEIGHT    = 2; //权重

function clog()
{

}

function on_connect(ws)
{
    // 打包set_pro数据包
    // 2字节cmd
    var r = "";
    r += (CMD_SET_PRO + "").charCodeAt();
    r += ((CMD_SET_PRO >> 8) + "").charCodeAt();

    // 4字节权重
    r += (weight + "").charCodeAt();
    r += ((weight >> 8) + "").charCodeAt();
    r += ((weight >> 16) + "").charCodeAt();
    r += ((weight >> 32) + "").charCodeAt();

    // 实际的分组名称
    r += group_name;

    console.log(r)
    ws.send(r)
}

function on_message(msg)
{
    console.log(msg)
    //cmd(2byte) + content
}

function on_close()
{

}

function on_error()
{

}

function start_service()
{
    var ws = new WebSocket("ws://127.0.0.1:9997/");

    ws.onopen = function() {
        on_connect(ws);
    };

    ws.onmessage = function(e) {
        on_message(e.data);
    };

    ws.onclose = function() {
        on_close();
    };

    ws.onerror = function(e) {
        on_error();
    };
}


start_service();
