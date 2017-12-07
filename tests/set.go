package main

import "fmt"

func main()  {
	set := []string{"Length","Height","Width","Weight","Color"}
	v := uint(17)

	res := ""
	for i:=0;i<len(set) ;i++  {
		fmt.Println((v & (1 << uint(i))), 1 << uint(i))
		if (v & (1 << uint(i))) > 0 {
			if res != "" {
				res += ","
			}
			res += set[i]
		}
	}

	fmt.Println(res)
}
