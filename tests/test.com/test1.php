<?php
/**
 * Created by PhpStorm.
 * User: yuyi
 * Date: 2017/11/13
 * Time: 21:47
 * 广播http接收端1
 */
 $data = file_get_contents("php://input");
 $arr = json_decode($data, true);
file_put_contents(__DIR__."/test1.log",
date("Y-m-d H:i:s"). "=>". json_encode($arr)."\r\n",
FILE_APPEND);
file_put_contents(__DIR__."/test11.log",$data);

echo "ok";