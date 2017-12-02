package services

type tcpGroupConfig struct {
    Mode int     // "1 broadcast" ##(广播)broadcast or  2 (权重)weight
    Name string  // = "group1"
    Filter []string
}
type tcpConfig struct {
    Listen string
    Port int
}
type TcpConfig struct {
    Enable bool
    Groups map[string]tcpGroupConfig
    Tcp tcpConfig
}

type HttpConfig struct {
    Enable bool
    Groups map[string]httpNodeConfig
}

type httpNodeConfig struct {
    Mode int
    Nodes [][]string
    Filter []string
}
