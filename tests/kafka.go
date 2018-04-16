package main

import (
	"github.com/Shopify/sarama"
	"time"
	"os"
	"os/signal"
	"fmt"
)
func main() {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForLocal       // Only wait for the leader to ack
	config.Producer.Compression = sarama.CompressionSnappy   // Compress messages
	config.Producer.Flush.Frequency = 500 * time.Millisecond // Flush batches every 500ms
	config.Version = sarama.V1_1_0_0

	retention := "-1"
	topicName := "testtopic9"
	req := &sarama.CreateTopicsRequest{
		//Version:sarama.V1_1_0_0,
		TopicDetails: map[string]*sarama.TopicDetail{
			topicName: {
				//---注意这里的三个参数，特别是最后一个不能与前面两个共存
				//这个参数必须要大于0
				NumPartitions:     -1,
				//这个设置不能大于当前集群内的节点数
				ReplicationFactor:3,//3,// -1,
				ReplicaAssignment: map[int32][]int32{0: []int32{0, 1, 2}},

				ConfigEntries: map[string]*string{
					"retention.ms": &retention,
				},
			},
		},
		Timeout: 100 * time.Millisecond,
	}
	broker := sarama.NewBroker("127.0.0.1:9092")
	defer broker.Close()
	broker.Open(config)
	r, e := broker.CreateTopics(req)
	fmt.Printf("%+v, %+v\n", r, e)
	fmt.Printf("%+v\n", *r.TopicErrors[topicName])

	cli, _ := sarama.NewClient([]string{"127.0.0.1:9092"}, config)
	defer cli.Close()
	t, _ := cli.Topics()
	fmt.Printf("topics: %v, %+v\n\n",len(t), t)

	//broker2 := sarama.NewBroker("127.0.0.1:9093")
	//broker2.Open(config)
	dreq := &sarama.DeleteTopicsRequest{
		Version:1,
		Topics:  []string{topicName},
		Timeout:time.Second,// time.Duration
	}
	d,e := broker.DeleteTopics(dreq)

	fmt.Printf("%+v, %+v\n", d,e)

	t, _ = cli.Topics()
	// here print the result, is same like before delete
	fmt.Printf("topics: %v, %+v\n\n",len(t), t)

	cli2, _ := sarama.NewClient([]string{"127.0.0.1:9092"}, config)
	defer cli.Close()
	//if here use the cli to get topics, the result is return same like before  delete
	//but new a cli, and get the topics again, is return ok, why?
	t, _ = cli2.Topics()
	// here is ok
	fmt.Printf("topics: %v, %+v\n\n",len(t), t)

	//wing-binlog-event
	//broker.AddOffsetsToTxn(
	//
	//)
	//broker.AddPartitionsToTxn()
	//broker.AlterConfigs()
	areq := &sarama.AddPartitionsToTxnRequest{
		TransactionalID: "txn",
		ProducerID:      8000,
		ProducerEpoch:   0,
		TopicPartitions: map[string][]int32{
			"wing-binlog-event": []int32{3},
		},
	}
	r1, e1 := broker.AddPartitionsToTxn(areq)
	fmt.Printf("%+v, %+v \n\n", *r1.Errors["wing-binlog-event"][0], e1)
	//broker.AlterConfigs()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	<-signals
}
