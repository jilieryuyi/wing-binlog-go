package app

import "errors"
var (
	ErrorFileNotFound = errors.New("文件不存在")
	ErrorFileParse = errors.New("配置解析错误")
)
