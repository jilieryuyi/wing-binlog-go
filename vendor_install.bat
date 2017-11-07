@echo off
set current_path=%cd%
set vendor_path=%current_path%\vendor\src

set mysql_path=%vendor_path%\mysql
set config_path=%vendor_path%\config

if exist %mysql_path% (
 rd %mysql_path% /S /Q
)

if not exist %mysql_path% (
 md %mysql_path%
)

if exist %config_path% (
 rd %config_path% /S /Q
)

if not exist %config_path% (
 md %config_path%
)

call git clone https://github.com/go-sql-driver/mysql.git %mysql_path%
rd %mysql_path%\.git  /S /Q

call git clone https://github.com/larspensjo/config.git %config_path%
rd %config_path%\.git  /S /Q