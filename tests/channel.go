package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"sync"
)

type S1 struct {
	lock *sync.Mutex
	ch chan int
}


//var c chan int = make(chan int, 3)
//var lock = new(sync.Mutex)
func (s1 *S1) test() {
	for {
		select {
		case i := <-s1.ch:
			log.Println("read=>", i)
		}
	}
}

func (s1 *S1) write() {
	for i := 0; i < 3; i++ {
		log.Println("write=>", i)
		s1.ch <- i
	}
}

func main() {
	var s1 = &S1{
		lock :new(sync.Mutex),
		ch:make(chan int, 3),
	}
	go s1.test()
	s1.write()
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		os.Kill,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	<-sc
}
