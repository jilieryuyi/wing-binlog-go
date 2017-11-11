package services

type tcpg struct {
    Mode int   // "1 broadcast" ##(广播)broadcast or  2 (权重)weight
    Name string// = "group1"
}
type tcpc struct {
    Listen string
    Port int
}
type TcpConfig struct {
    Groups map[string]tcpg
    Tcp tcpc
}
