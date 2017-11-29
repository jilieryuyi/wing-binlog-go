package buffer

import (
	"errors"
)

var out_of_range = errors.New("out of range")

type WBuffer struct {
	buf []byte
	pos int
	size int
}

func NewBuffer(size int) *WBuffer {
	return &WBuffer{buf:make([]byte, size), pos:0, size:0}
}

func (wb *WBuffer) Write(w []byte, size int) {
	wb.buf = append(wb.buf[:wb.size], w...)
	wb.size += size
}

func (wb *WBuffer) Size() int {
	return wb.size - wb.pos
}

func (wb *WBuffer) Len() int {
	return len(wb.buf)
}

func (wb *WBuffer) Read(n int) ([]byte, error) {
	if wb.size - wb.pos <= 0 {
		return nil, out_of_range
	}
	b := wb.buf[wb.pos:wb.pos + n]
	wb.pos += n
	return b, nil
}

func (wb *WBuffer) ResetPos()  {
	wb.buf = append(wb.buf[:0], wb.buf[wb.pos:wb.size]...)
	wb.size -= wb.pos
	wb.pos = 0
}

func (wb *WBuffer) ReadInt32() (int, error) {
	if wb.size - wb.pos < 4 {
		return 0, out_of_range
	}
	b , err := wb.Read(4)
	i := int(b[0]) + int(b[1] << 8) + int(b[2] << 16) + int(b[3] << 32)
	return i, err
}

func (wb *WBuffer) ReadInt16() (int,error) {
	if wb.size - wb.pos < 2 {
		return 0, out_of_range
	}
	b, err := wb.Read(2)
	i := int(b[0]) + int(b[1] << 8)
	return i, err
}
