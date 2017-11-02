@echo off
set current_path=%cd%
set library_ip_path=%current_path%\src\library\ip
set library_debug_path=%current_path%\src\library\debug
set library_path=%current_path%\src\library
set ini_test_file=%current_path%\src\config\mysql.ini

set GOPATH=%current_path%\vendor;%current_path%

copy /y %ini_test_file% "C:\__test_mysql.ini"
echo 123 > C:\__test.txt
cd %~d0
cd %library_ip_path%
call go test
cd %library_debug_path%
call go test
cd %library_path%
call go test
del "C:\__test_mysql.ini"
del "C:\__test.txt"
cd %current_path%