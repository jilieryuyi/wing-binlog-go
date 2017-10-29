package tcp

import "fmt"

type Tcp struct {

}

func (r *Tcp) OnChange(data map[string] interface{}) {
	fmt.Println("tcp", data)
}
