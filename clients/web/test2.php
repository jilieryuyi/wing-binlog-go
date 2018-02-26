<?php
/**
 * Created by PhpStorm.
 * User: yuyi
 * Date: 2017/11/13
 * Time: 21:47
 * 广播http接收端2
 */
file_put_contents("test2.log", date("Y-m-d H:i:s"). "=>". file_get_contents("php://input")."\r\n",
    FILE_APPEND);