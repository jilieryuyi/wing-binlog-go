package main

import (
    "github.com/scottkiss/gosshtool"
    "fmt"
)

func main() {
    sshconfig := &gosshtool.SSHClientConfig{
        User:     "root",
        Password: "",
        Host:     "",
    }
    sshclient := gosshtool.NewSSHClient(sshconfig)
    fmt.Println(sshclient.Host)
    stdout, stderr,session, err := sshclient.Cmd("ls /",nil,nil,0)
    if err != nil {
        fmt.Println(err)
    }
    fmt.Println(session)
    fmt.Println(stdout)
    fmt.Println(stderr)
}
