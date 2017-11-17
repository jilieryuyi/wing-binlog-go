package main

import (
    "fmt"
    "golang.org/x/crypto/ssh"
    "net"
)

func main() {

    sshConfig := &ssh.ClientConfig{
        User: "root",
        Auth: []ssh.AuthMethod{ssh.Password("")},
        //需要验证服务端，不做验证返回nil就可以，点击HostKeyCallback看源码就知道了
        HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
            return nil
        },
    }

    client, err := ssh.Dial("tcp", "", sshConfig)
    if err != nil {
        fmt.Println(err)
        return
    }

    session, err := client.NewSession()
    if err != nil {
        fmt.Println(err)
        client.Close()
        return
    }


    if err != nil {
        panic(err)
    }
    out, err := session.CombinedOutput("ls /home")
    if err != nil {
        fmt.Println(err)
    }
    fmt.Println(string(out))
    client.Close()

}
