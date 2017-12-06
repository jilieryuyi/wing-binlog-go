package services

import (
	kafka "github.com/segmentio/kafka-go"
	"context"
	"sync"
	log "github.com/sirupsen/logrus"
)

func NewKafkaService() *WKafka {
	config, _:= getKafkaConfig()
	log.Debugf("kafka服务配置：%+v", config)
	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  config.Borkers,
		/* []string {
			"172.16.214.194:9092",
			"172.16.214.195:9092",
			"172.16.214.196:9092",
		},*/
		Topic:config.Topic,//    "wing-binlog-go",
		Balancer: &kafka.LeastBytes{},
	})
	return &WKafka {
		writer     : w,
		is_closed  : false,
		lock       : new(sync.Mutex),
		send_queue : make(chan []byte, TCP_MAX_SEND_QUEUE),
		enable     : config.Enable,
	}
}

func (wk *WKafka) SendAll(msg []byte) bool {
	log.Info("kafka服务-发送消息")
	if len(wk.send_queue) >= cap(wk.send_queue) {
		log.Warn("kafka服务-发送缓冲区满")
		return false
	}
	wk.send_queue <- msg
	return true
}

func (wk *WKafka) Start() {
	if !wk.enable {
		return
	}
	go func() {
		for {
			if wk.is_closed {
				log.Info("kafka服务关闭")
				return
			}
			select {
			case  msg, ok := <- wk.send_queue:
				if !ok {
					log.Info("kafka服务-发送消息channel通道关闭")
					return
				}
				table_len := int(msg[0]) + int(msg[1] << 8)
				event := msg[table_len + 2:]
				table := msg[2:table_len + 2]
				log.Debugf("kafka服务-发送消息：%s %s", table, event)
				err := wk.writer.WriteMessages(context.Background(),
					kafka.Message{
						Key: table,
						Value: event,
					},
				)
				if err != nil {
					log.Errorf("kafka服务-发送消息错误：%+v", err)
				}
			}
		}
	} ()
}

func (wk *WKafka) Close()  {
	wk.lock.Lock()
	wk.is_closed = true
	wk.writer.Close()
	wk.lock.Unlock()
}
