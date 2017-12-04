package data

import (
	"testing"
)

func TestUser_Add(t *testing.T) {
	u := User{"yuyi", "123456"}
	u.Add()
}