package squid

import (
	"fmt"
	"net"
)

const (
	LOCALHOST = "0.0.0.0"
)

func GetFreePort() (port uint16, e error) {
	var a *net.TCPAddr
	if a, e = net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", LOCALHOST)); e == nil {
		var l *net.TCPListener
		if l, e = net.ListenTCP("tcp", a); e == nil {
			defer l.Close()
			return uint16(l.Addr().(*net.TCPAddr).Port), nil
		}
	}
	return port, e
}
