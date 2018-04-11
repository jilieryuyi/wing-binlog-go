package kafka

import (
	"library/services"
	log "github.com/sirupsen/logrus"
	"github.com/Shopify/sarama"
)

type Producer struct {
	services.Service
	AccessLogProducer sarama.AsyncProducer
	enable bool
	topic string
	filter []string
}

var _ services.Service = &Producer{}

func NewProducer() services.Service {
	config, _ := getConfig()
	if !config.Enable {
		return &Producer{
			enable:false,
		}
	}
	//brokers := strings.Split(",", config.Brokers)
	//[]string{"127.0.0.1:9092"}),
	log.Debugf("kafka config: %+v", *config)
	return &Producer{
		AccessLogProducer:newAccessLogProducer(config.Brokers),
		enable:true,
		topic:config.Topic,
		filter:config.Filter,
	}
}

func (r *Producer) SendAll(table string, data []byte) bool {
	if !r.enable {
		return false
	}
	entry := &accessLogEntry{
		Data:data,
	}

	if services.MatchFilters(r.filter, table) {
		return false
	}
	// We will use the client's IP address as key. This will cause
	// all the access log entries of the same IP address to end up
	// on the same partition.
	r.AccessLogProducer.Input() <- &sarama.ProducerMessage{
		Topic: r.topic,
		Key:   sarama.StringEncoder(table),
		Value: entry,
	}
	return true
}
func (r *Producer) Start() {}
func (r *Producer) Close() {
	if !r.enable {
		return
	}
	if err := r.AccessLogProducer.Close(); err != nil {
		log.Println("Failed to shut down access log producer cleanly", err)
	}
}
func (r *Producer) Reload() {
	config, _ := getConfig()
	r.AccessLogProducer = newAccessLogProducer(config.Brokers)
	r.enable = true
	r.topic = config.Topic
	r.filter = config.Filter
}
func (r *Producer) Name() string {
	return "kafka"
}
func (r *Producer) SendRaw(data []byte) bool { return true }


