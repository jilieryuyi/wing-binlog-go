package services

import (
    "io/ioutil"
    "fmt"
    "net/http"
    "strings"
    "net/url"
    "encoding/json"
    "bytes"
    "net"
    "time"
)
type HttpService struct {
    urls []string
}
func httpDo(json string) {
    client := &http.Client{}

    req, err := http.NewRequest("POST", "http://www.01happy.com/demo/accept.php", strings.NewReader("name=cjb"))
    if err != nil {
        // handle error
    }

    req.SetBasicAuth("root", "123456")
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.Header.Set("Cookie", "name=anny")
    req.Form = url.Values{"event":{json}}
    //req.
    resp, err := client.Do(req)

    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        // handle error
    }

    fmt.Println(string(body))
}

func NetUploadJson(addr string, buf interface{}) (*[]byte, *int, error) {
    // 将需要上传的JSON转为Byte
    v, _ := json.Marshal(buf)
    // 上传JSON数据
    req, e := http.NewRequest("POST", addr, bytes.NewReader(v))
    if e != nil {
        // 提交异常,返回错误
        return nil, nil, e
    }
    // Body Type
    req.Header.Set("Content-Type", "application/json")
    // 完成后断开连接
    req.Header.Set("Connection", "close")
    // -------------------------------------------
    // 设置 TimeOut
    DefaultClient := http.Client{
        Transport: &http.Transport{
            Dial: func(netw, addr string) (net.Conn, error) {
                deadline := time.Now().Add(30 * time.Second)
                c, err := net.DialTimeout(netw, addr, time.Second*30)
                if err != nil {
                    return nil, err
                }
                c.SetDeadline(deadline)
                return c, nil
            },
        },
    }
    // -------------------------------------------
    // 执行
    resp, ee := DefaultClient.Do(req)
    if ee != nil {
        // 提交异常,返回错误
        return nil, nil, ee
    }
    // 保证I/O正常关闭
    defer resp.Body.Close()
    // 判断返回状态
    if resp.StatusCode == http.StatusOK {
        // 读取返回的数据
        data, err := ioutil.ReadAll(resp.Body)
        if err != nil {
            // 读取异常,返回错误
            return nil, nil, err
        }
        // 将收到的数据与状态返回
        return &data, &resp.StatusCode, nil
    } else if resp.StatusCode != http.StatusOK {
        // 返回异常状态
        return nil, &resp.StatusCode, nil
    }
    // 不会到这里
    return nil, nil, nil
}

// 下载文件
func NetDownloadFile(addr string) (*[]byte, *int, *http.Header, error) {
    // 上传JSON数据
    req, e := http.NewRequest("GET", addr, nil)
    if e != nil {
        // 返回异常
        return nil, nil, nil, e
    }
    // 完成后断开连接
    req.Header.Set("Connection", "close")
    // -------------------------------------------
    // 设置 TimeOut
    DefaultClient := http.Client{
        Transport: &http.Transport{
            Dial: func(netw, addr string) (net.Conn, error) {
                deadline := time.Now().Add(30 * time.Second)
                c, err := net.DialTimeout(netw, addr, time.Second*30)
                if err != nil {
                    return nil, err
                }
                c.SetDeadline(deadline)
                return c, nil
            },
        },
    }
    // -------------------------------------------
    // 执行
    resp, ee := DefaultClient.Do(req)
    if ee != nil {
        // 返回异常
        return nil, nil, nil, ee
    }
    // 保证I/O正常关闭
    defer resp.Body.Close()
    // 判断请求状态
    if resp.StatusCode == 200 {
        data, err := ioutil.ReadAll(resp.Body)
        if err != nil {
            // 读取错误,返回异常
            return nil, nil, nil, err
        }
        // 成功，返回数据及状态
        return &data, &resp.StatusCode, &resp.Header, nil
    } else {
        // 失败，返回状态
        return nil, &resp.StatusCode, nil, nil
    }
    // 不会到这里
    return nil, nil, nil, nil
}
