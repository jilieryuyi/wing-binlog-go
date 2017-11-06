package subscribe

import (
	"encoding/json"
	"library/base"
	"log"
	"runtime"
)

type Redis struct {
	base.Subscribe
	queue chan map[string]interface{}
}

func (r *Redis) Init() {
	r.queue = make(chan map[string]interface{}, base.MAX_EVENT_QUEUE_LENGTH)

	//to := time.NewTimer(time.Second*3)
	cpu := runtime.NumCPU()
	for i := 0; i < cpu; i++ {
		go func() {
			for {
				select {
				case body := <-r.queue:
					for k, v := range body {
						log.Println("redis---", k, v)
					}

					json_byte, _ := json.Marshal(body)
					log.Println(string(json_byte))
					//case <-to.C://time.After(time.Second*3):
					//	Log("发送超时...")
				}
			}
		}()
	}
}

func (r *Redis) OnChange(data map[string]interface{}) {
	r.queue <- data
	log.Println("redis", data)
}

func (r *Redis) Free() {
	close(r.queue)
}
