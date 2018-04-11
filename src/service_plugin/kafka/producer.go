package kafka

import (
	"library/services"
	log "github.com/sirupsen/logrus"
	"github.com/Shopify/sarama"
	"time"
)

type Producer struct {
	services.Service
	AccessLogProducer sarama.AsyncProducer
}

var _ services.Service = &Producer{}

func NewProducer() services.Service {
	return &Producer{
		AccessLogProducer:newAccessLogProducer([]string{"127.0.0.1:9092"}),
	}
}

func (r *Producer) SendAll(table string, data []byte) bool {
	entry := &accessLogEntry{
		Data:data,
	}

	// We will use the client's IP address as key. This will cause
	// all the access log entries of the same IP address to end up
	// on the same partition.
	r.AccessLogProducer.Input() <- &sarama.ProducerMessage{
		Topic: "wing-binlog-event",
		Key:   sarama.StringEncoder(table),
		Value: entry,
	}
	return true
}
func (r *Producer) Start() {}
func (r *Producer) Close() {
	if err := r.AccessLogProducer.Close(); err != nil {
		log.Println("Failed to shut down access log producer cleanly", err)
	}
}
func (r *Producer) Reload() {
}
func (r *Producer) Name() string {
	return "kafka"
}
func (r *Producer) SendRaw(data []byte) bool { return true }

func newAccessLogProducer(brokerList []string) sarama.AsyncProducer {

	// For the access log, we are looking for AP semantics, with high throughput.
	// By creating batches of compressed messages, we reduce network I/O at a cost of more latency.
	config := sarama.NewConfig()
	//tlsConfig := createTlsConfiguration()
	//if tlsConfig != nil {
	//	config.Net.TLS.Enable = true
	//	config.Net.TLS.Config = tlsConfig
	//}
	config.Producer.RequiredAcks = sarama.WaitForLocal       // Only wait for the leader to ack
	config.Producer.Compression = sarama.CompressionSnappy   // Compress messages
	config.Producer.Flush.Frequency = 500 * time.Millisecond // Flush batches every 500ms

	producer, err := sarama.NewAsyncProducer(brokerList, config)
	if err != nil {
		log.Fatalln("Failed to start Sarama producer:", err)
	}

	// We will just log to STDOUT if we're not able to produce messages.
	// Note: messages will only be returned here after all retry attempts are exhausted.
	go func() {
		for err := range producer.Errors() {
			log.Println("Failed to write access log entry:", err)
		}
	}()

	return producer
}


type accessLogEntry struct {
	Data []byte
	encoded []byte
	err error
}

func (ale *accessLogEntry) ensureEncoded() {
	//if ale.encoded == nil && ale.err == nil {
	ale.encoded = ale.Data//json.Marshal(ale)
	//}
}

func (ale *accessLogEntry) Length() int {
	ale.ensureEncoded()
	return len(ale.encoded)
}

func (ale *accessLogEntry) Encode() ([]byte, error) {
	ale.ensureEncoded()
	return ale.encoded, ale.err
}


