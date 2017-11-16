package main

import (
    "regexp"
    "fmt"
)
func main(){
    fmt.Println(regexp.MatchString(`xsl\.*`, "xsl.123"));
}
