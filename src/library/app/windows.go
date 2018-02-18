// +build  !linux,!darwin,windows

//https://studygolang.com/articles/4207
package app

import (
	"library/file"
	//"bitbucket.org/kardianos/service"
	//"github.com/kardianos/service"
	//"fmt"
	//"os"
	//https://github.com/zuiwuchang/windows-go-daemon/blob/master/go-daemon/main.go
)

func DaemonProcess(d bool) bool {
	return false
}

func Release() {
	// delete pid when exit
	file.Delete(Pid)
}

//var log service.Logger
//func DaemonProcess(d bool) bool {
//	var name = "wing-binlog-go"
//	var displayName = "wing-binlog-go -daemon -config-path=" +ConfigPath
//	var desc = "wing binlog daemon process"
//
//	var s, err = service.NewService(name, displayName, desc)
//	//log = s
//	if err != nil {
//		fmt.Printf("%s unable to start: %s", displayName, err)
//		return true
//	}
//	if len(os.Args) > 1 {
//		var err error
//		verb := os.Args[1]
//		switch verb {
//		case "install":
//			err = s.Install()
//			if err != nil {
//				fmt.Printf("Failed to install: %s\n", err)
//				return
//			}
//			fmt.Printf("Service \"%s\" installed.\n", displayName)
//		case "remove":
//			err = s.Remove()
//			if err != nil {
//				fmt.Printf("Failed to remove: %s\n", err)
//				return
//			}
//			fmt.Printf("Service \"%s\" removed.\n", displayName)
//		case "run":
//			doWork()
//		case "start":
//			err = s.Start()
//			if err != nil {
//				fmt.Printf("Failed to start: %s\n", err)
//				return
//			}
//			fmt.Printf("Service \"%s\" started.\n", displayName)
//		case "stop":
//			err = s.Stop()
//			if err != nil {
//				fmt.Printf("Failed to stop: %s\n", err)
//				return
//			}
//			fmt.Printf("Service \"%s\" stopped.\n", displayName)
//		}
//		return
//	}
//	err = s.Run(func() error {
//		// start
//		go doWork()
//		return nil
//	}, func() error {
//		// stop
//		stopWork()
//		return nil
//	})
//	if err != nil {
//		s.Error(err.Error())
//	}
//	return false
//}
//func doWork() {
//	log.Info("I'm Running!")
//	select {}
//}
//func stopWork() {
//	log.Info("I'm Stopping!")
//}