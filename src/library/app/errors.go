package app

import "errors"

var (
	ErrorFileNotFound = errors.New("file does not exists")
	ErrorFileParse    = errors.New("config file parse error")
)
