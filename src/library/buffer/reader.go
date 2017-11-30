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

// 创建一个buffer，size指定是初始化缓存长度
func NewBuffer(size int) *WBuffer {
	return &WBuffer{buf:make([]byte, size), pos:0, size:0}
}

// 写入byte
func (wb *WBuffer) Write(w []byte) {
	wb.buf = append(wb.buf[:wb.size], w...)
	wb.size += len(w)
}

// 剩余的未读的元素长度
func (wb *WBuffer) Size() int {
	return wb.size - wb.pos
}

// buf的实际长度
func (wb *WBuffer) Len() int {
	return len(wb.buf)
}

// 读取指定字节
func (wb *WBuffer) Read(n int) ([]byte, error) {
	if wb.size - wb.pos <= 0 {
		return nil, out_of_range
	}
	b := wb.buf[wb.pos:wb.pos + n]
	wb.pos += n
	return b, nil
}

// 清理掉前面已读的部分，也就是删除数组的元素
func (wb *WBuffer) ResetPos()  {
	wb.buf = append(wb.buf[:0], wb.buf[wb.pos:wb.size]...)
	wb.size -= wb.pos
	wb.pos = 0
}

// 读取int32，int32占4个字节，即解包int32
func (wb *WBuffer) ReadInt32() (int, error) {
	if wb.size - wb.pos < 4 {
		return 0, out_of_range
	}
	b , err := wb.Read(4)
	i := int(b[0]) + int(b[1] << 8) + int(b[2] << 16) + int(b[3] << 32)
	return i, err
}

// 读取int16，int16占2个字节，即解包int16
func (wb *WBuffer) ReadInt16() (int,error) {
	if wb.size - wb.pos < 2 {
		return 0, out_of_range
	}
	b, err := wb.Read(2)
	i := int(b[0]) + int(b[1] << 8)
	return i, err
}
