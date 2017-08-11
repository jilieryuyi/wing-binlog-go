package main

import "database/sql"
import (
	_ "github.com/go-sql-driver/mysql"
	"fmt"
)

func main() {

	defer func(){ // 必须要先声明defer，否则不能捕获到panic异常
		if err:=recover();err!=nil{
			fmt.Println("发生错误\r\n")
			fmt.Println(err) // 这里的err其实就是panic传入的内容，55
		}
		fmt.Println("进程结束\r\n")
	}()

	user     := "root"
	password := "123456"
	db, err  := sql.Open(
		"mysql",
		user+":"+ password+"@tcp(127.0.0.1:3306)/xl?charset=utf8")

	if (nil != err) {
		panic(err);
		return;
	}
	stmt, err := db.Prepare("INSERT INTO content_type(name, sort, status, img, created_at, updated_at, real_name) VALUES(?,?,?,?,?,?,?)")
	if (nil != err) {
		panic(err);
		return;
	}
	res, err := stmt.Exec("yuyi", 20, 1, "http://www.xunlei.com/", 0,0,"yuyi")
	if (nil != err) {
		panic(err);
		return;
	}
	id, err := res.LastInsertId()
	if (nil != err) {
		panic(err);
		return;
	}
	fmt.Println(id)
}