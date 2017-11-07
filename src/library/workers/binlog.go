package workers

import (
	"fmt"
	"library/base"
	"time"
)

type Binlog struct {
	base.Worker
}

/**
 * 初始化订阅者
 * @param []base.Subscribe notify
 */
func (log *Binlog) Start(notify []base.Subscribe) {
	for _, obj := range notify {
		obj.Init()
	}
}

/**
 * 清理资源，放在main函数的defer里面就可以了
 * @param []base.Subscribe notify
 */
func (log *Binlog) End(notify []base.Subscribe) {
	for _, obj := range notify {
		obj.Free()
	}
}

/**
 * 标准的事件监听流程
 */
func (log *Binlog) Loop(notify []base.Subscribe) {

	data := make(map[string]interface{})
	data["hello2"] = "yuyi2"

	for {
		for _, obj := range notify {
			obj.OnChange(data)
		}
		time.Sleep(1000000000)
	}
	fmt.Println(notify)
}
