package binlog

import (
	"unicode/utf8"
	//"golang.org/x/text/encoding/simplifiedchinese"
	//"golang.org/x/text/transform"
	//"io/ioutil"
	//"bytes"
	//"github.com/axgle/mahonia"
	"library/path"
	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
	"library/file"
)

// 字符串编码
func encode(buf *[]byte, s string) {
	*buf = append(*buf, '"')
	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if htmlSafeSet[b] {
				i++
				continue
			}
			if start < i {
				*buf = append(*buf, s[start:i]...)
			}
			switch b {
			case '\\', '"':
				*buf = append(*buf, '\\')
				*buf = append(*buf, b)
			case '\n':
				*buf = append(*buf, '\\')
				*buf = append(*buf, 'n')
			case '\r':
				*buf = append(*buf, '\\')
				*buf = append(*buf, 'r')
			case '\t':
				*buf = append(*buf, '\\')
				*buf = append(*buf, 't')
			default:
				// This encodes bytes < 0x20 except for \t, \n and \r.
				// If escapeHTML is set, it also escapes <, >, and &
				// because they can lead to security holes when
				// user-controlled strings are rendered into JSON
				// and served to some browsers.
				*buf = append(*buf, `\u00`...)
				*buf = append(*buf, hex[b>>4])
				*buf = append(*buf, hex[b&0xF])
			}
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRuneInString(s[i:])
		if c == utf8.RuneError && size == 1 {
			if start < i {
				*buf = append(*buf, s[start:i]...)
			}
			*buf = append(*buf, `\ufffd`...)
			i += size
			start = i
			continue
		}
		// U+2028 is LINE SEPARATOR.
		// U+2029 is PARAGRAPH SEPARATOR.
		// They are both technically valid characters in JSON strings,
		// but don't work in JSONP, which has to be evaluated as JavaScript,
		// and can lead to security holes there. It is valid JSON to
		// escape them, so we do so unconditionally.
		// See http://timelessrepo.com/json-isnt-a-javascript-subset for discussion.
		if c == '\u2028' || c == '\u2029' {
			if start < i {
				*buf = append(*buf, s[start:i]...)
			}
			*buf = append(*buf, `\u202`...)
			*buf = append(*buf, hex[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		*buf = append(*buf, s[start:]...)
	}
	*buf = append(*buf, '"')
}

//func (h *binlogHandler) toUtf8(str string) []byte {
//	data, _ := ioutil.ReadAll(
//		transform.NewReader(bytes.NewReader([]byte(str)),
//			simplifiedchinese.GBK.NewEncoder()))
//	return data
//}

//func toUtf8(str string) string {
//	enc:=mahonia.NewEncoder("utf8")
//	//converts a  string from UTF-8 to gbk encoding.
//	return enc.ConvertString(str)
//}

// 封包
func pack(cmd int, client_id string, msgs []string) []byte {
	client_id_len := len(client_id)
	// 获取实际包长度
	l := 0
	for _, msg := range msgs {
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
		m := []byte(msg)
		ml := len(m)
		// 前4字节存放长度
		r[base_start+0] = byte(ml)
		r[base_start+1] = byte(ml >> 8)
		r[base_start+2] = byte(ml >> 16)
		r[base_start+3] = byte(ml >> 24)
		base_start += 4
		// 实际的内容
		copy(r[base_start:], m)
		base_start += ml
	}
	return r
}

func packPos(binFile string, pos int64, eventIndex int64) []byte {
	res := []byte(binFile)
	l := 16 + len(res)
	r := make([]byte, l + 2)
	// 2 bytes is data length
	r[0] = byte(l)
	r[1] = byte(l >> 8)
	// 8 bytes is pos
	r[2] = byte(pos)
	r[3] = byte(pos >> 8)
	r[4] = byte(pos >> 16)
	r[5] = byte(pos >> 24)
	r[6] = byte(pos >> 32)
	r[7] = byte(pos >> 40)
	r[8] = byte(pos >> 48)
	r[9] = byte(pos >> 56)
	// 8 bytes is event index
	r[10] = byte(eventIndex)
	r[11] = byte(eventIndex >> 8)
	r[12] = byte(eventIndex >> 16)
	r[13] = byte(eventIndex >> 24)
	r[14] = byte(eventIndex >> 32)
	r[15] = byte(eventIndex >> 40)
	r[16] = byte(eventIndex >> 48)
	r[17] = byte(eventIndex >> 56)
	// the last is binlog file
	r = append(r[:18], res...)
	return r
}

func unpackPos(data []byte) (string, int64, int64) {
	pos := int64(data[2]) | int64(data[3])<<8 | int64(data[4])<<16 |
		int64(data[5])<<24 | int64(data[6])<<32 | int64(data[7])<<40 |
		int64(data[8])<<48 | int64(data[9])<<56
	eventIndex := int64(data[10]) | int64(data[11])<<8 |
		int64(data[12])<<16 | int64(data[13])<<24 |
		int64(data[14])<<32 | int64(data[15])<<40 |
		int64(data[16])<<48 | int64(data[17])<<56
	return string(data[18:]), pos, eventIndex
}

func getConfig() (*Config, error) {
	var config Config
	configFile := path.CurrentPath + "/config/cluster.toml"
	wfile := file.WFile{configFile}
	if !wfile.Exists() {
		log.Errorf("config file not found: %s", configFile)
		return nil, ErrorFileNotFound
	}
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		log.Println(err)
		return nil, ErrorFileParse
	}
	return &config, nil
}

