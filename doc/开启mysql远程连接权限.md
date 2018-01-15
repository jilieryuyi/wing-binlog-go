如果是5.7版本，按需要执行以下语句，用来设置mysql弱密码，比如设置123456时会失败，执行以下语句解决此问题
错误：Your password does not satisfy the current policy requirements
解决：
````
SET GLOBAL  validate_password_policy='LOW';
````
开启权限：
````
GRANT ALL PRIVILEGES ON *.* TO 'root'@'%' IDENTIFIED BY '123456' WITH GRANT OPTION;
````
权限生效：
````
FLUSH PRIVILEGES;
````
如果还是无法连接，确认一下本地mysql监听的ip是否为127.0.0.1，如果是，修改为0.0.0.0即可，
一般情况下，配置文件路径如下：
/etc/my.conf
/usr/local/etc/my.conf
修改如下：（mac示例，修改文件/usr/local/etc/my.conf）
````
 # Default Homebrew MySQL server config
 [mysqld]
 # Only allow connections from localhost
 bind-address = 0.0.0.0
````
重启mysql，执行：
````
brew services restart mysql
````