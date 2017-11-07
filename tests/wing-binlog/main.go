package main

import "fmt"

//import (
//	"fmt"
//	"os"
//	"os/exec"
//	"path/filepath"
//)

import (
	"./Library"
	//"./String"
)

func main() {

	defer func() { // 必须要先声明defer，否则不能捕获到panic异常
		if err := recover(); err != nil {
			fmt.Println("发生错误\r\n")
			fmt.Println(err) // 这里的err其实就是panic传入的内容，55
		}
		fmt.Println("进程结束\r\n")
	}()

	//fmt.Println(os.Getppid())
	//if os.Getppid() != 1 {
	//	//判断当其是否是子进程，当父进程return之后，子进程会被 系统1 号进程接管
	//	filePath, _ := filepath.Abs(os.Args[0])
	//	//将命令行参数中执行文件路径转换成可用路径
	//	cmd := exec.Command(filePath)
	//	//将其他命令传入生成出的进程
	//	cmd.Stdin = os.Stdin
	//	//给新进程设置文件描述符，可以重定向到文件中
	//	cmd.Stdout = os.Stdout
	//	cmd.Stderr = os.Stderr
	//	//开始执行新进程，不等待新进程退出
	//	cmd.Start()
	//	return
	//}

	pdo := Library.Pdo{"root", "123456", "xl"}
	pdo.Open()
	defer pdo.Close()

	binlog := Library.Binlog{}
	fmt.Println(binlog.GetLogs(), "\r\n\r\n")

	//str := fmt.Sprintf("%f", 0.123456);
	fmt.Println(binlog.GetFormat(), "\r\n\r\n")
	fmt.Println(binlog.GetCurrentLogInfo(), "\r\n\r\n")
	//fmt.Println("hello_"+str);
}
