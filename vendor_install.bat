@echo off
set current_path=%cd%
set vendor_path=%current_path%\vendor

::添加当前目录和当前目录下的vendor目录到GOPATH环境变量
set GOPATH=%current_path%\vendor;%current_path%;%GOPATH%

if not exist %vendor_path% (
 md %vendor_path%
 md %vendor_path%\src
)

call go get github.com/go-sql-driver/mysql
call go get github.com/larspensjo/config
call go get github.com/siddontang/go-mysql/canal
call go get github.com/siddontang/go-mysql/replication
call go get github.com/siddontang/go-mysql/mysql
call go get github.com/BurntSushi/toml
call go get golang.org/x/crypto/ssh/terminal

xcopy  %vendor_path%\src\*.* %vendor_path% /s /e
