package ip

import (
    "net"
    "errors"
)

func Local() (string, error) {
    ifaces, err := net.Interfaces()
    if err != nil {
        return "", err
    }

    var ip net.IP

    for _, iface := range ifaces {
        if iface.Flags&net.FlagUp == 0 {
            continue // interface down
        }
        if iface.Flags&net.FlagLoopback != 0 {
            continue // loopback interface
        }
        addrs, err := iface.Addrs()
        if err != nil {
            return "", err
        }
        for _, addr := range addrs {
            switch v := addr.(type) {
                case *net.IPNet:
                    ip = v.IP
                case *net.IPAddr:
                    ip = v.IP
            }
            if ip == nil || ip.IsLoopback() {
                continue
            }
            ip = ip.To4()
            if ip == nil {
                continue // not an ipv4 address
            }
            return ip.String(), nil
        }
    }
    return "", errors.New("are you connected to the network?")
}

/**
func GetLocalIP() string {
    addrs, err := net.InterfaceAddrs()
    if err != nil {
        return ""
    }
    for _, address := range addrs {
        // check the address type and if it is not a loopback the display it
        if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
            if ipnet.IP.To4() != nil {
                return ipnet.IP.String()
            }
        }
    }
    return ""
}
*/

/**
import (
    "log"
    "net"
    "strings"
)

// Get preferred outbound ip of this machine
func GetOutboundIP() net.IP {
    conn, err := net.Dial("udp", "8.8.8.8:80")
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    localAddr := conn.LocalAddr().(*net.UDPAddr)

    return localAddr.IP
}
*/

