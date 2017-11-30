package cluster

// 封包
func pack(cmd int, client_id string, msgs []string) []byte {
	client_id_len := len(client_id)
	// 获取实际包长度
	l := 0
	for _,msg := range msgs {
		l += len([]byte(msg)) + 4
	}
	// cl为实际的包内容长度，2字节cmd
	cl := l + 2 + client_id_len
	r := make([]byte, cl)
	r[0] = byte(cmd)
	r[1] = byte(cmd >> 8)
	copy(r[2:], []byte(client_id))
	base_start := 2 + client_id_len
	for _, msg := range msgs {
		m  := []byte(msg)
		ml := len(m)
		// 前4字节存放长度
		r[base_start + 0] = byte(ml)
		r[base_start + 1] = byte(ml >> 8)
		r[base_start + 2] = byte(ml >> 16)
		r[base_start + 3] = byte(ml >> 32)
		base_start += 4
		// 实际的内容
		copy(r[base_start:], m)
		base_start += ml
	}
	return r
}

