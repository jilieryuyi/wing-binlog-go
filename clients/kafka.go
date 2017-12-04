package main

import (
	kafka "github.com/segmentio/kafka-go"
	"context"
	"fmt"
)

func main() {
	fmt.Println("start...")
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{"172.16.214.194:9092", "172.16.214.195:9092", "172.16.214.196:9092"},
		Topic:     "wing-binlog-go",
		Partition: 0,
		MinBytes:  10e3, // 10KB
		MaxBytes:  10e6, // 10MB
	})
	r.SetOffset(0)
	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			break
		}
		fmt.Printf("message at offset %d: %s = %s\n", m.Offset, string(m.Key), string(m.Value))
	}
	r.Close()
}
