@echo off
set current_path=%cd%
set vendor_path=%current_path%\vendor

::添加当前目录和当前目录下的vendor目录到GOPATH环境变量
set GOPATH=%vendor_path%;%current_path%

if not exist %vendor_path% (
 md %vendor_path%
 md %vendor_path%\src
)

echo installing... go-sql-driver/mysql
call go get github.com/go-sql-driver/mysql
echo installing... larspensjo/config
call go get github.com/larspensjo/config
echo installing... siddontang/go-mysql/canal
call go get github.com/siddontang/go-mysql/canal
echo installing... siddontang/go-mysql/replication
call go get github.com/siddontang/go-mysql/replication
echo installing... siddontang/go-mysql/mysql
call go get github.com/siddontang/go-mysql/mysql
echo installing... BurntSushi/toml
call go get github.com/BurntSushi/toml
echo installing... go-martini/martini
call go get github.com/go-martini/martini
echo installing... gorilla/websocket
call go get github.com/gorilla/websocket
echo installing... garyburd/redigo/redis
call go get github.com/garyburd/redigo/redis
echo "installing... takama/daemon"
call go get github.com/takama/daemon
echo "installing... mattn/go-sqlite3"
call go get github.com/mattn/go-sqlite3
echo "installing... segmentio/kafka-go"
call go get github.com/segmentio/kafka-go
echo "installing... golang.org/x/text/encoding/simplifiedchinese"
call go get golang.org/x/text/encoding/simplifiedchinese
echo "installing... golang.org/x/text/transform"
call go get golang.org/x/text/transform
echo "installing... github.com/axgle/mahonia"
call go get github.com/axgle/mahonia
echo "installing... github.com/hashicorp/consul"
call go get github.com/hashicorp/consul
echo "installing... github.com/go-redis/redis"
call go get github.com/go-redis/redis
echo "installing... github.com/Shopify/sarama"
call go get github.com/Shopify/sarama
echo "installing... github.com/sirupsen/logrus"
call go get github.com/sirupsen/logrus

::xcopy  %vendor_path%\src\*.* %vendor_path% /s /e /y /q
