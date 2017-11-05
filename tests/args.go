package main

import "fmt"

func add(a ...int) int {
    sum := 0
    for _,v := range a  {
        fmt.Println(v)
        sum += v
    }
    return sum
}
func main() {
    nums := []int{1,2,3,4}
    sum := add(nums...);
    fmt.Println(sum)
}
