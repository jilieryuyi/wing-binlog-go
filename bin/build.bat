@echo off
echo building...

set pan=%~d0
set current_path=%cd%
set bin_path=%current_path%\bin
set pkg_path=%current_path%\pkg
set vendor_path=%current_path%\vendor

::添加环境变量,即在原来的环境变量后加上英文状态下的分号和路径
set GOPATH=%vendor_path%;%current_path%

::如果pkg目录存在，直接删除
if exist %pkg_path% (
 rd %pkg_path% /S /Q
 )

::如果vendor目录不存在，创建该目录
if not exist %vendor_path% (
 md %vendor_path%
 md %vendor_path%\src
 call "%current_path%\vendor_install.bat"
)

::cd %pan%
::进入当前目录
::cd %current_path%
::build构建项目
call go build -p 4 -race  wing-binlog-go
::install安装
call go install wing-binlog-go
::删除根目录下的可执行文件
del %current_path%\wing-binlog-go.exe

if not exist %bin_path%\config (
md %bin_path%\config
)

::拷贝配置文件
xcopy  %current_path%\src\config\*.* %current_path%\bin\config\ /s /e /y /q
echo build success
echo %current_path%\bin\wing-binlog-go.exe