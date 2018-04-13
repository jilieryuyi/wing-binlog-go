package main

import (
"fmt"
"log"
"os"
"os/signal"

cluster "github.com/bsm/sarama-cluster"
)

func main() {

	// init (custom) config, enable errors and notifications
	config := cluster.NewConfig()
	config.Consumer.Return.Errors = true
	config.Group.Return.Notifications = true

	// init consumer
	brokers := []string{"127.0.0.1:9092", "127.0.0.1:9093", "127.0.0.1:9094"}
	topics := []string{"wing-binlog-event"}
	c,_:=cluster.NewClient(brokers,config)
	bb := c.Brokers()
	fmt.Printf("Brokers: %+v\n", bb)
	for _, v:=range bb {
		fmt.Printf("Broker: %+v\n", *v)
	}

	tt, _ := c.Topics()
	fmt.Printf("Topics: %+v\n", tt)

	consumer, err := cluster.NewConsumer(brokers, "my-consumer-group", topics, config)
	if err != nil {
		panic(err)
	}
	defer consumer.Close()

	go func () {
		for {
			select {
				case pp, ok := <-consumer.Partitions():
					if !ok {
						return
					}
					fmt.Printf("Partition: %+v", pp)
			}
		}
	}()

	// trap SIGINT to trigger a shutdown.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	// consume errors
	go func() {
		for err := range consumer.Errors() {
			log.Printf("Error: %s\n", err.Error())
		}
	}()

	// consume notifications
	go func() {
		for ntf := range consumer.Notifications() {
			log.Printf("Rebalanced: %+v\n", ntf)
		}
	}()

	count := 0
	// consume messages, watch signals
	for {
		select {
		case msg, ok := <-consumer.Messages():
			if ok {
				count++
				fmt.Fprintf(os.Stdout, "%d ====> %s/%d/%d\t%s\t%s\n",count, msg.Topic, msg.Partition, msg.Offset, msg.Key, msg.Value)
				consumer.MarkOffset(msg, "")	// mark message as processed
			}
		case <-signals:
			return
		}
	}
}


