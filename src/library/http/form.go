package http

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/http"
	"time"
	log "github.com/jilieryuyi/logrus"
	"errors"
	"fmt"
)

type Http struct {
	url string
}

func NewHttp(url string) *Http {
	request := &Http{
		url : url,
	}
	return request
}

// 超时设置
var defaultHttpClient = http.Client{
	Transport: &http.Transport{
		MaxIdleConnsPerHost: 16,
		Dial: func(netw, addr string) (net.Conn, error) {
			dial := net.Dialer{
				Timeout:   HTTP_POST_TIMEOUT * time.Second,
				KeepAlive: 86400 * time.Second,
			}
			conn, err := dial.Dial(netw, addr)
			if err != nil {
				return conn, err
			}
			return conn, nil
		},
	},
}

func Post(addr string, postData []byte) ([]byte, error) {
	c := NewHttp(addr)
	return c.Post(postData)
}

func Get(addr string) ([]byte, error) {
	c := NewHttp(addr)
	return c.Get()
}

func request(method string, url string, data []byte)  ([]byte, error) {
	reader := bytes.NewReader(data)
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		log.Errorf("syshttp request error-1:%+v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "keep-alive")
	resp, err := defaultHttpClient.Do(req)
	if err != nil {
		log.Errorf("http request error-3:%+v", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Errorf("http request error错误的状态码:%+v", resp.StatusCode)
		return nil, errors.New(fmt.Sprintf("错误的状态码：%d", resp.StatusCode))
	}
	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("http request error-4:%+v", err)
		return nil, err
	}
	return res, nil
}

func (req *Http) Put(data []byte) ([]byte, error) {
	return request("PUT", req.url, data)
}

func (req *Http) Post(data []byte) ([]byte, error) {
	log.Debugf("post: %s", string(data))
	return request("POST", req.url, data)
}

func (req *Http) Get() ([]byte, error) {
	return request("GET", req.url, nil)
}

func (req *Http) Delete() ([]byte, error) {
	return request("DELETE", req.url, nil)
}

