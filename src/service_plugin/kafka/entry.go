package kafka

type accessLogEntry struct {
	Data []byte
}

func (ale *accessLogEntry) Length() int {
	return len(ale.Data)
}

func (ale *accessLogEntry) Encode() ([]byte, error) {
	return ale.Data, nil
}



