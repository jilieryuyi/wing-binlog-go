package std

import (
    "testing"
    "fmt"
)

func TestReset(t *testing.T)  {
    err := Reset()

    if err != nil {
        fmt.Println(err)
    }
}
