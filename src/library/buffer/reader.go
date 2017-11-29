package buffer

import "github.com/siddontang/go-mysql/client"

type WBuffer struct {
	buf []byte
	pos int
	size int
}

func NewBuffer() *WBuffer {
	return &WBuffer{buf:make([]byte, 4096), pos:0, size:0}
}

func (wb *WBuffer) Write(w []byte) {
	wb.buf = append(wb.buf[:wb.size], w...)
	wb.size += len(w)
}

func (wb *WBuffer) Read(n int) []byte {
	b := wb.buf[wb.pos:wb.pos + n]
	wb.pos += n
	return b
}

func (wb *WBuffer) ResetPos()  {
	wb.buf = append(wb.buf[:0], wb.buf[wb.pos:wb.size]...)
	wb.size -= wb.pos
	wb.pos = 0
}

func (wb *WBuffer) ReadIn32() int {
	b := wb.Read(4)
	i := int(b[0]) +
		int(b[1] << 8) +
		int(b[2] << 16) +
		int(b[3] << 32)
	return i
}
