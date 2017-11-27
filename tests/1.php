<?php
/**
 * Created by PhpStorm.
 * User: yuyi
 * Date: 17/11/10
 * Time: 22:55
 */
//4 0 0 0
$len = (4) +
    (0 << 8) +
    (0 << 16) +
    (0 << 32);


var_dump($len);

while(1) {
    echo file_get_contents("http://localhost:9989/user/login?username=admin&password=admin");
    //usleep(100000);
}
