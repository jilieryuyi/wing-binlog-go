package http

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/http"
	"time"
	"errors"
	"fmt"
)

const (
	httpDefaultTimeout = 30   //30秒超时
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

var defaultHttpClient = http.Client{
	Transport: &http.Transport{
		MaxIdleConnsPerHost: 16,
		Dial: func(netw, addr string) (net.Conn, error) {
			dial := net.Dialer{
				Timeout:   httpDefaultTimeout * time.Second,
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
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "keep-alive")
	resp, err := defaultHttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("error http status：%d", resp.StatusCode))
	}
	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (req *Http) Put(data []byte) ([]byte, error) {
	return request("PUT", req.url, data)
}

func (req *Http) Post(data []byte) ([]byte, error) {
	return request("POST", req.url, data)
}

func (req *Http) Get() ([]byte, error) {
	return request("GET", req.url, nil)
}

func (req *Http) Delete() ([]byte, error) {
	return request("DELETE", req.url, nil)
}

