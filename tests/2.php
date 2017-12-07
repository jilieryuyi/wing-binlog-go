<?php
$str = file_get_contents(__DIR__."/test.com/test11.log");
echo $str,"\r\n\r\n";
$arr = json_decode($str, true);
//foreach ($arr["event"]["data"]["old_data"] as &$v) {
//    $v = urldecode($v);
//}
//foreach ($arr["event"]["data"]["new_data"] as &$v) {
//    $v = urldecode($v);
//}
var_dump($arr);