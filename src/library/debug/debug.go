package debug

import (
	"fmt"
	"time"
	//"os"
	//"library/path"
)

func Print(a ...interface{}) {
	fmt.Print(time.Now().Format("2006-01-02 15:04:05"))
	fmt.Print(" ");
	for _,v := range a  {
		fmt.Print(v);
		fmt.Print(" ");
	}
	fmt.Print("\n");
}
