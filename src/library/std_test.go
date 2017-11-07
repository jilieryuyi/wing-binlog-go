package library

import (
	"fmt"
	"testing"
)

func TestReset(t *testing.T) {
	err := Reset()

	if err != nil {
		fmt.Println(err)
	}
}
