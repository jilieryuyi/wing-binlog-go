//事件计数器
var count         = 1;
//事件
const CMD_SET_PRO = 1;
const CMD_AUTH    = 2;
const CMD_OK      = 3;
const CMD_ERROR   = 4;
const CMD_TICK    = 5;
const CMD_EVENT   = 6;
const CMD_RELOGIN = 8;

//心跳包
var tick_pack = String.fromCharCode(CMD_TICK) + String.fromCharCode(CMD_TICK >> 8);
error_times = 0;

function ws_connect() {
    return new WebSocket("ws://127.0.0.1:9988/");
}

//获取当前的时间，格式为 Y-m-d H:i:s
function get_daytime() {
    //获取时间
    var myDate = new Date();
    var month = myDate.getMonth() + 1;
    month = month < 10 ? "0" + month : month;
    var day = myDate.getDate();
    day = day < 10 ? "0" + day : day;
    var hour = myDate.getHours();
    hour = hour < 10 ? "0" + hour : hour;
    var minute = myDate.getMinutes();
    minute = minute < 10 ? "0" + minute : minute;
    var seconds = myDate.getSeconds();
    seconds = seconds < 10 ? "0" + seconds : seconds;
    return myDate.getFullYear() + '-' + month + '-' + day + ' ' + hour + ':' + minute + ":" + seconds;
}

//打印debug信息
function clog(content) {
    var len = arguments.length;
    var cc = get_daytime() + " ", i = 0;
    for (i = 0; i < len; i++) {
        cc += arguments[i] + " ";
    }
    console.log(cc);
}

//连接成功回调函数
function on_connect() {
    connect_status = 2;
    // str="A";
    // code = str.charCodeAt();
    // 打包set_pro数据包
    // 2字节cmd
    var r = "";
    r += String.fromCharCode(CMD_AUTH);
    r += String.fromCharCode(CMD_AUTH >> 8);

    var user_sign = Cookies.get('user_sign');
    if (!user_sign) {
        return;
        //alert("连接出错");
    }
    // 签名
    r += user_sign;

    //连接成功后发送注册到分组事件
    ws.send(r)
}

//收到消息回调函数
function on_message(msg) {
    var cmd = msg[0].charCodeAt() + msg[1].charCodeAt();
    var content = msg.substr(2, msg.length - 2);

    switch (cmd) {
        case CMD_SET_PRO:
            clog("设置注册分组返回值：", content);
            break;
        case CMD_AUTH:
            clog("连接成功");
            break;
        case CMD_OK:
            clog("正常响应返回值：", content);
            break;
        case CMD_ERROR:
            clog("错误返回值：", content);
            break;
        case CMD_TICK:
            clog("心跳返回值：", content);
            break;
        case CMD_EVENT:
            clog("事件返回值：", count, content);
            var div = document.createElement("div");
            div.innerHTML = get_daytime() + "<br/>第" + count + "次收到事件<br/>" + content + "<br/>";
            document.body.appendChild(div);
            count++;
            break;
        case CMD_RELOGIN:
            window.location.href="login.html";
            break;
        default:
            clog("未知事件：", cmd, content);
    }
}

//客户端关闭回调函数
function on_close() {
    clog("客户端断线，尝试重新连接");
    error_times++;
    // window.setTimeout(function () {
    //     ws = ws_connect();
    //     start_service();
    // }, 3000);
}

//发生错误回调函数
function on_error() {
    clog("客户端发生错误，尝试重新连接");
    error_times++;

}

//开始服务
function start_service() {
    ws.onopen = function () {
        on_connect();
    };

    ws.onmessage = function (e) {
        on_message(e.data);
    };

    ws.onclose = function () {
        on_close();
    };

    ws.onerror = function (e) {
        on_error();
    };
}


var user_sign = Cookies.get('user_sign');
if (!user_sign) {
    window.location.href="/login.html";
} else {
    var ws = ws_connect();
    start_service();
    //5秒发送一次心跳
    window.setInterval(function () {
        ws.send(tick_pack);
    }, 5000);
    window.setInterval(function () {
        if (error_times > 0) {
            ws = ws_connect();
            start_service();
            error_times = 0;
        }
    }, 3000);
}
