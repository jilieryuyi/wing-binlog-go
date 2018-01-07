package library

import (
	"fmt"
	"library/file"
	"log"
	"os"
	"sync"
	"time"
)

func Reset() error {
	dir := file.GetCurrentPath() + "/logs"
	logs_dir := &file.WPath{dir}

	if !logs_dir.Exists() {
		os.Mkdir(dir, 0755)
	}

	stdout_log_path := fmt.Sprintf(dir+"/stdout-%d.log", time.Now().Unix())
	handle_stdout, err := os.OpenFile(stdout_log_path, os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0755)
	os.Stdout = handle_stdout

	if err != nil {
		return err
	}

	stderr_log_path := fmt.Sprintf(dir+"/stderr-%d.log", time.Now().Unix())
	handle_stderr, err := os.OpenFile(stderr_log_path, os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0755)
	os.Stderr = handle_stderr

	if err != nil {
		return err
	}

	go func() {
		var mutex sync.Mutex
		for {
			s, err := os.Stderr.Stat()
			if err != nil {
				time.Sleep(time.Second)
				continue
			}
			if s.Size() == 1024*1024*1024*1024 {
				// == 1G
				mutex.Lock()
				os.Stdout.Close()
				os.Stderr.Close()

				stdout_log_path := fmt.Sprintf(dir+"/stdout-%d.log", time.Now().Unix())
				handle_stdout, err := os.OpenFile(stdout_log_path, os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0755)
				os.Stdout = handle_stdout

				if err != nil {
					log.Println(err)
					mutex.Unlock()
					time.Sleep(time.Second)
					continue
				}

				stderr_log_path := fmt.Sprintf(dir+"/stderr-%d.log", time.Now().Unix())
				handle_stderr, err := os.OpenFile(stderr_log_path, os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0755)
				os.Stderr = handle_stderr

				if err != nil {
					log.Println(err)
					mutex.Unlock()
					time.Sleep(time.Second)
					continue
				}

				mutex.Unlock()
			}
			time.Sleep(time.Second)
		}
	}()

	return nil
}
