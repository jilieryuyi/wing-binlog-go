package main

import (
    "os"
    "fmt"
)

func main() {
    _, err := os.Stat("/")
    if err != nil {
        fmt.Print("error", err)
    }

    if os.IsNotExist(err) {
        fmt.Print("/ not extists")
    }


}
