var count      = 0;
var group_name = "group1";
var weight     = 100;
var ws         = new WebSocket("ws://127.0.0.1:9997/");


const CMD_SET_PRO = 1;
const CMD_AUTH    = 2;
const CMD_OK      = 3;
const CMD_ERROR   = 4;
const CMD_TICK    = 5;
const CMD_EVENT   = 6;

const MODE_BROADCAST = 1; //广播
const MODE_WEIGHT    = 2; //权重

var tick_pack  = String.fromCharCode(CMD_TICK) + String.fromCharCode(CMD_TICK >> 8);

function clog()
{

}

function on_connect()
{
    // str="A";
    // code = str.charCodeAt();
    // 打包set_pro数据包
    // 2字节cmd
    var r = "";
    r +=  String.fromCharCode(CMD_SET_PRO);
    r +=  String.fromCharCode(CMD_SET_PRO >> 8);

    // 4字节权重
    r +=  String.fromCharCode(weight);
    r +=  String.fromCharCode(weight >> 8);
    r +=  String.fromCharCode(weight >> 16);
    r +=  String.fromCharCode(weight >> 32);

    // 实际的分组名称
    r += group_name;

    ws.send(r)
}

function on_message(msg)
{
    var cmd     = msg[0].charCodeAt() + msg[1].charCodeAt();
    var content = msg.substr(2, msg.length - 2);

    switch (cmd) {
        case CMD_SET_PRO:
            console.log("设置注册分组返回值：", content);
            break;
        case CMD_AUTH:
        case CMD_OK:
            console.log("正常响应返回值：", content);
            break;
        case CMD_ERROR:
            console.log("错误返回值：", content);
            break;
        case CMD_TICK:
            console.log("心跳返回值：", content);
            break;
        case CMD_EVENT:
            console.log("事件返回值：", content);
            break;
        default:
            console.log("未知事件：", cmd, content);
    }
}

function on_close()
{

}

function on_error()
{

}

function start_service()
{

    ws.onopen = function() {
        on_connect();
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
window.setInterval(function(){
   ws.send(tick_pack);
}, 3000);
