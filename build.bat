@echo off
set current_path=%cd%
::echo %current_path%

set bin_path=%current_path%\bin
set pkg_path=%current_path%\pkg
set vendor_path=%current_path%\vendor

::echo %bin_path%
::echo %pkg_path%
::echo %vendor_path%

::如果bin目录存在，直接删除掉
if exist %bin_path% (
 rd %bin_path% /S /Q
 )

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
