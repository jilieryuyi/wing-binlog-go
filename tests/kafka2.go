package main

import (
"fmt"
"os"
"os/signal"

cluster "github.com/bsm/sarama-cluster"
	"github.com/Shopify/sarama"
	"time"
)

func main() {

	// init (custom) config, enable errors and notifications
	config := cluster.NewConfig()
	config.Consumer.Return.Errors = true
	config.Group.Return.Notifications = true

	// init consumer
	brokers := []string{"127.0.0.1:9092", "127.0.0.1:9093", "127.0.0.1:9094"}
	//topics := []string{"wing-binlog-event"}
	c,_:=cluster.NewClient(brokers,config)
	bb := c.Brokers()
	fmt.Printf("Brokers: %+v\n", bb)
	for _, v:=range bb {
		fmt.Printf("Broker: %+v\n", *v)
	}


	retention := "-1"
	req := &sarama.CreateTopicsRequest{
		//Version:sarama.V1_1_0_0,
		TopicDetails: map[string]*sarama.TopicDetail{
			"testtopic": {
				NumPartitions:     3,
				ReplicationFactor: 3,
				ReplicaAssignment: map[int32][]int32{
					0: []int32{0, 1, 2},
				},
				ConfigEntries: map[string]*string{
					"retention.ms": &retention,
				},
			},
		},
		Timeout: 100 * time.Millisecond,
	}

	bb[0].Open(nil)
	r, e := bb[0].CreateTopics(req)
	fmt.Printf("%+v, %+v\n", r, e)

	tt, _ := c.Topics()
	fmt.Printf("Topics: %+v\n", tt)



	// trap SIGINT to trigger a shutdown.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)




	<-signals
}



