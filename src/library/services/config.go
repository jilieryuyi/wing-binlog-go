package services

type tcpg struct {
    Name string  // = "group1"
    Filter []string
}
type tcpc struct {
    Listen string
    Port int
}
type TcpConfig struct {
    Enable bool
    Groups map[string]tcpg
    Tcp tcpc
}

type HttpConfig struct {
    Enable bool
    Groups map[string]httpNodeConfig
}

type httpNodeConfig struct {
    Nodes [][]string
    Filter []string
}
