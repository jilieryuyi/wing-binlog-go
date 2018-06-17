package subscribe

func hasCmd(cmd int) bool {
	return cmd == CMD_SET_PRO ||
		cmd == CMD_AUTH ||
		cmd == CMD_ERROR||
		cmd == CMD_TICK ||
		cmd == CMD_EVENT||
		cmd == CMD_AGENT||
		cmd == CMD_STOP||
		cmd == CMD_RELOAD||
		cmd == CMD_SHOW_MEMBERS||
		cmd == CMD_POS
}

//func PackPro(flag int, content []byte) []byte {
//	// 数据打包
//	l := len(content) + 3
//	//log.Debugf("PackPro len: %d", l)
//	r := make([]byte, l + 4)
//	// 4字节数据包长度
//	r[0] = byte(l)
//	r[1] = byte(l >> 8)
//	r[2] = byte(l >> 16)
//	r[3] = byte(l >> 24)
//	// 2字节cmd
//	r[4] = byte(CMD_SET_PRO)
//	r[5] = byte(CMD_SET_PRO >> 8)
//	r[6] = byte(flag)
//	// 实际数据内容
//	r = append(r[:7], content...)
//	return r
//}
