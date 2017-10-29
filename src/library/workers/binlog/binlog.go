package binlog

import (
	"library/base"
	"fmt"
)

type Binlog struct {

}

func (log *Binlog) Loop(notify []base.Subscribe) {

	data := make(map[string] interface{})
	data["hello2"] = "yuyi2"

	for _,obj := range notify {
		obj.OnChange(data);
	}
	fmt.Println(notify)
}
