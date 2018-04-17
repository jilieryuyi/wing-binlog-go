<?php
/**
 * Created by PhpStorm.
 * User: yuyi
 * Date: 2017/11/13
 * Time: 21:47
 * 80%权重接收端
 */
file_put_contents("test3.log", date("Y-m-d H:i:s"). "=>". file_get_contents("php://input")."\r\n",
    FILE_APPEND);