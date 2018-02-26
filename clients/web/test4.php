<?php
/**
 * Created by PhpStorm.
 * User: yuyi
 * Date: 2017/11/13
 * Time: 21:48
 * 20%权重接收端
 */
file_put_contents("test4.log", date("Y-m-d H:i:s"). "=>". file_get_contents("php://input")."\r\n",
    FILE_APPEND);