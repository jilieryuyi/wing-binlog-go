package main

import (
	"runtime"
	"path"
	"fmt"
	"time"
)
func c() {
	pc, _file, line, ok := runtime.Caller(8)
	fmt.Println(pc, _file, line, ok)
	if ok {
		funcName := runtime.FuncForPC(pc).Name()
		fmt.Println(path.Base(_file))
		fmt.Println(path.Base(funcName))
		fmt.Println(line)
	}
}

func b() {
	pc := make([]uintptr, 3, 3)
	cnt := runtime.Callers(6, pc)
	fmt.Printf("\n\n====%+v====\n\n", cnt)
	for i := 0; i < cnt; i++ {
		fu := runtime.FuncForPC(pc[i] - 1)
		_file, line := fu.FileLine(pc[i] - 1)
		fmt.Printf("\n\n====%+v====%s, %s, %d\n\n", fu, fu.Name(), _file, line)
		fu = runtime.FuncForPC(pc[i] - 2)
		_file, line = fu.FileLine(pc[i] - 2)
		fmt.Printf("\n\n====%+v====%s, %s, %d\n\n", fu, fu.Name(), _file, line)
		fu = runtime.FuncForPC(pc[i] - 3)
		_file, line = fu.FileLine(pc[i] - 2)
		fmt.Printf("\n\n====%+v====%s, %s, %d\n\n", fu, fu.Name(), _file, line)

		name := fu.Name()
			_file, line = fu.FileLine(pc[i] - 1)
			fmt.Println(path.Base(_file))
			fmt.Println(path.Base(name))
			fmt.Println(line)
	}
	c()
}

func a() {
	b()
	for{
		time.Sleep(time.Second*3)
	}
}
