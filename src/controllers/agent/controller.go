package agent

import (
	"github.com/jilieryuyi/wing-go/tcp"
	"library/service"
	"github.com/sirupsen/logrus"
	"encoding/json"
)

type Controller struct {
	onEvent []OnEventFunc
	onPos []OnPosFunc
}
type OnEventFunc func(table string, data []byte)
type OnPosFunc func(data []byte)
type Option func(controller *Controller)

const (
	CMD_SET_PRO = iota // 注册客户端操作，加入到指定分组
	CMD_AUTH           // 认证（暂未使用）
	CMD_ERROR          // 错误响应
	CMD_TICK           // 心跳包
	CMD_EVENT          // 事件
	CMD_AGENT
	CMD_STOP
	CMD_RELOAD
	CMD_SHOW_MEMBERS
	CMD_POS
)

func SetOnEvent(f OnEventFunc) Option {
	return func(controller *Controller) {
		controller.onEvent = append(controller.onEvent, f)
	}
}

func SetOnPos(f OnPosFunc) Option {
	return func(controller *Controller) {
		controller.onPos = append(controller.onPos, f)
	}
}

func NewController(options ...Option) *Controller {
	c := &Controller{
		onEvent: make([]OnEventFunc, 0),
		onPos: make([]OnPosFunc, 0),
	}
	for _, f := range options {
		f(c)
	}
	return c
}

func (c *Controller) onMessage(client *tcp.Client, content []byte) {
	cmd, data, err := service.Unpack(content)
	if err != nil {
		logrus.Error(err)
		return
	}
	switch cmd {
	case CMD_EVENT:
		var raw map[string] interface{}
		err = json.Unmarshal(data, &raw)
		if err == nil {
			table := raw["database"].(string) + "." + raw["table"].(string)
			for _, f := range c.onEvent  {
				f(table, data)
			}
		}
	case CMD_POS:
		for _, f := range c.onPos  {
			f(data)
		}
	}
}
