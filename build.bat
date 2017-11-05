@echo off
set current_path=%cd%
::echo %current_path%

set bin_path=%current_path%\bin
set pkg_path=%current_path%\pkg
set vendor_path=%current_path%\vendor

::echo %bin_path%
::echo %pkg_path%
::echo %vendor_path%

::添加环境变量,即在原来的环境变量后加上英文状态下的分号和路径
set GOPATH=%current_path%\vendor;%current_path%

::如果bin目录存在，直接删除掉
::if exist %bin_path% (
:: rd %bin_path% /S /Q
:: )

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

cd %~d0
::进入当前目录
cd %current_path%
::build构建项目
call go build wing-binlog-go
::install安装
call go install wing-binlog-go
::删除根目录下的可执行文件
del %current_path%\wing-binlog-go.exe

if not exist %bin_path%\config (
md %bin_path%\config
)

::拷贝配置文件
xcopy  %current_path%\src\config\*.* %current_path%\bin\config\ /s /e
echo "build success"