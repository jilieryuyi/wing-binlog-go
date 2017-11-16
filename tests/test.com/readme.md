http协议测试，对应配置文件wing-binlog-go/src/config/http.toml的测试    
apache简单配置

<VirtualHost *:80>
DocumentRoot /Users/yuyi/Code/go/wing-binlog-go/tests/test.com
ServerName test.com
</VirtualHost>

<Directory "/Users/yuyi/Code/go/wing-binlog-go/tests/test.com">
    Options Indexes FollowSymLinks Includes ExecCGI
    AllowOverride All
    Require all granted
</Directory>
