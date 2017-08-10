package main

import "database/sql"
import (
	_ "github.com/go-sql-driver/mysql"
	"fmt"
	//"reflect"
	//"strconv"
	"reflect"
	"encoding/json"
	"./String"
)


func main() {

	defer func(){ // 必须要先声明defer，否则不能捕获到panic异常
		if err:=recover();err!=nil{
			fmt.Println("发生错误\r\n")
			fmt.Println(err) // 这里的err其实就是panic传入的内容，55
		}
		fmt.Println("进程结束\r\n")
	}()

	res := query("select * from content_type where id=?", 2)
	fmt.Println(res)

	str, _ := json.Marshal(res)

	fmt.Println(string(str))

	kk :=12456

	vv := String.Convert{kk}

	fmt.Println("hello_"+vv.ToString())

	//if (nil != err) {
	//	panic(err);
	//	return;
	//}
	//id, err := res.LastInsertId()
	//if (nil != err) {
	//	panic(err);
	//	return;
	//}
	//fmt.Println(id)
}

func query(_sql string, args ...interface{})  map[int]map[string]interface {} {
	user     := "root"
	password := "123456"
	db, err  := sql.Open(
		"mysql",
		user+":"+ password+"@tcp(127.0.0.1:3306)/xl?charset=utf8")

	if (nil != err) {
		panic(err);
		return nil;
	}
	rows, err := db.Query(_sql, args...)
	if (nil != err) {
		panic(err);
		return nil;
	}
	defer rows.Close();
	//fmt.Println(rows);
	//records := make(map[int]map[string]string)
	//index:=0;
	//for rows.Next() {
	//	record := make(map[string]string)
	//	var id int
	//	var name string
	//	if err := rows.Scan(&id,&name); nil == err {
	//		fmt.Printf("%d == %s \r\n", id, name)
	//		record["id"] = strconv.Itoa(id)
	//		record["name"] = name
	//	}
	//	records[index] = record;
	//	fmt.Println(record);
	//	index = index+1
	//}
	//fmt.Println(records);

	columns, _ := rows.Columns()
	//fmt.Println(columns)

	scanArgs := make([]interface{}, len(columns))
	values := make([]interface{}, len(columns))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	//fmt.Println(scanArgs)
	records := make(map[int]map[string]interface {})
	index :=0
	for rows.Next() {
		//将行数据保存到record字典
		err = rows.Scan(scanArgs...)
		//fmt.Println(values)
		record := make(map[string]interface {})
		for i, col := range values {
			if col != nil {
				fmt.Printf("%s的类型是：%s\r\n", columns[i], reflect.TypeOf(col))
				//fmt.Println(reflect.TypeOf(col))
				switch col.(type) { //多选语句switch
				case string:
					record[columns[i]] = string(col.([]byte))
					//是字符时做的事情
					//fmt.Printf("%s是string=%s\r\n", columns[i], col)
				case []uint8:
					record[columns[i]] = string(col.([]byte))
					//是字符时做的事情
					//fmt.Printf("%s是string222=%s\r\n", columns[i], col)

				case int:
					//是整数时做的事情
					//fmt.Printf("%s是int=%d\r\n", columns[i], col)
					record[columns[i]] = int(col.(int));//strconv.Itoa(int(col.(int)))
				case int64:
					//fmt.Printf("%s是int64=%d\r\n", columns[i], col)
					//_id := int64(col.([]int64));
					record[columns[i]] = int64(col.(int64));//strconv.FormatInt(int64(col.(int64)),10)
				}

				//record[columns[i]] = string(col.([]byte))

			}
		}
		records[index] = record
		fmt.Println(record)
		index++

	}
	fmt.Println(records)
return records
}