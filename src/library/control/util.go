package control

func pack(cmd int, msg []byte) []byte {
	//m  := []byte(msg)
	l  := len(msg)
	r  := make([]byte, l+6)
	cl := l + 2
	r[0] = byte(cl)
	r[1] = byte(cl >> 8)
	r[2] = byte(cl >> 16)
	r[3] = byte(cl >> 24)
	r[4] = byte(cmd)
	r[5] = byte(cmd >> 8)
	copy(r[6:], msg)
	return r
}

//func unpack(data []byte) (int, string) {
//	clen := int(data[0]) | int(data[1]) << 8 |
//		int(data[2]) << 16 | int(data[3]) << 24
//	cmd  := int(data[4]) | int(data[5]) << 8
//	content := string(data[6 : clen + 4])
//	return cmd, content
//}

func hasCmd(cmd int) bool {
	return cmd == CMD_ERROR||
		cmd == CMD_TICK ||
		cmd == CMD_STOP||
		cmd == CMD_RELOAD||
		cmd == CMD_SHOW_MEMBERS
}