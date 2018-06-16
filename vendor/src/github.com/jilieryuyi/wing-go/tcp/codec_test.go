package tcp

import (
	"testing"
	"bytes"
	"fmt"
)

func TestCodec_Encode(t *testing.T) {
	msgId := int64(1)
	data := []byte("hello")

	codec := &Codec{}
	cc := codec.Encode(msgId, data)

	mid, c, p, err := codec.Decode(cc)
	fmt.Println(mid, c, p, err)
	if err != nil {
		t.Errorf(err.Error())
	}
	if mid != msgId {
		t.Error("error")
	}

	if !bytes.Equal(c, data) {
		t.Error("error 2")
	}


	content := make([]byte, 0)
	content = append(content, []byte("你好")...)
	content = append(content, cc...)
	content = append(content, []byte("你好")...)
	content = append(content, cc...)
	content = append(content, []byte("qwrqwerfq34wfq")...)

	mid, c, p, err = codec.Decode(content)
	fmt.Println(mid, c, p, err)
	if err != nil {
		t.Errorf(err.Error())
	}
	if mid != msgId {
		t.Error("error")
	}

	if !bytes.Equal(c, data) {
		t.Error("error 2")
	}

	content = append(content[:0], content[p:]...)

	mid, c, p, err = codec.Decode(content)
	if err != nil {
		t.Errorf(err.Error())
	}
	if mid != msgId {
		t.Error("error")
	}

	if !bytes.Equal(c, data) {
		t.Error("error 2")
	}

	content = append(content[:0], content[p:]...)


	mid, c, p, err = codec.Decode(content)
	fmt.Println(mid, c, p, err)
	if err == nil {
		t.Errorf("error")
	}
	if mid == msgId {
		t.Error("error")
	}

	if bytes.Equal(c, data) {
		t.Error("error 2")
	}

}
