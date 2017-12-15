package http

import (
	"net/http"
	"bytes"
	"net"
	"time"
	"io/ioutil"
	//log "github.com/sirupsen/logrus"
	"errors"
	"fmt"
)

// 超时设置
var defaultHttpClient = http.Client {
	Transport: &http.Transport {
		Dial: func(netw, addr string) (net.Conn, error) {
			deadline := time.Now().Add(HTTP_POST_TIMEOUT * time.Second)
			c, err := net.DialTimeout(netw, addr, time.Second * HTTP_POST_TIMEOUT)
			if err != nil {
				return nil, err
			}
			c.SetDeadline(deadline)
			return c, nil
		},
	},
}

func Post(addr string, post_data []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", addr, bytes.NewReader(post_data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "Keep-Alive")
	// 执行post
	resp, err := defaultHttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	// 关闭io
	defer resp.Body.Close()
	// 判断返回状态
	if resp.StatusCode != http.StatusOK {
		// 返回异常状态
		//log.Errorf("http post error, error status back: %d", resp.StatusCode)
		return nil, errors.New(fmt.Sprintf("错误的状态码：%d", resp.StatusCode))//ErrorHttpStatus
	}
	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return res, nil
}
func Get(addr string) (*[]byte, *int, *http.Header, error) {
	// 上传JSON数据
	req, e := http.NewRequest("GET", addr, nil)
	if e != nil {
		// 返回异常
		return nil, nil, nil, e
	}
	// 完成后断开连接
	req.Header.Set("Connection", "close")
	// 执行
	resp, ee := defaultHttpClient.Do(req)
	if ee != nil {
		// 返回异常
		return nil, nil, nil, ee
	}
	// 保证I/O正常关闭
	defer resp.Body.Close()
	// 判断请求状态
	if resp.StatusCode == 200 {
		res, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			// 读取错误,返回异常
			return nil, nil, nil, err
		}
		// 成功，返回数据及状态
		return &res, &resp.StatusCode, &resp.Header, nil
	} else {
		// 失败，返回状态
		return nil, &resp.StatusCode, nil, nil
	}
	// 不会到这里
	return nil, nil, nil, nil
}

