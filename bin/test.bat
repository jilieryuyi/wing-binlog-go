@echo off
set current_path=%cd%
set library_ip_path=%current_path%\src\library\ip
set library_debug_path=%current_path%\src\library\debug
set library_path=%current_path%\src\library
set ini_test_file=%current_path%\src\config\mysql.ini

set GOPATH=%current_path%\vendor;%current_path%

cd %~d0
cd %current_path%\src
for /R %%s in (.,*) do (
if exist %%s\ (
cd %%s
call go test
)
)

::cd %~d0
::cd %library_ip_path%
::call go test
::cd %library_debug_path%
::call go test
::cd %library_path%
::call go test
cd %current_path%