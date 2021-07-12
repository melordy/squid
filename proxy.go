package squid

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/sirupsen/logrus"
)

type Proxy struct {
	lis       net.Listener
	port      uint16
	addr      string
	isClosed  bool
	onConnect func(net.Conn) error
}

func NewProxy(addr string) (*Proxy, error) {
	if len(addr) == 0 {
		return nil, errors.New("please provide the addr of the proxy server")
	}
	return &Proxy{
		addr: addr,
	}, nil
}

func (sp *Proxy) OnClientConnect(cb func(net.Conn) error) {
	if cb != nil {
		sp.onConnect = cb
	}
}

func (sp *Proxy) Start(port uint16) (e error) {
	sp.port = port
	if sp.port == 0 {
		sp.port, e = GetFreePort()
		if e != nil {
			return e
		}
	}
	sp.lis, e = net.Listen("tcp", fmt.Sprintf("%s:%d", LOCALHOST, sp.port))
	if e != nil {
		return e
	}
	go sp.accept()
	return nil
}

func (sp *Proxy) GetPort() uint16 {
	return sp.port
}

func (sp *Proxy) accept() {
	for {
		conn, e := sp.lis.Accept()
		if e != nil {
			if sp.isClosed {
				return
			}
			logrus.Error(e)
			continue
		}
		if sp.onConnect != nil {
			if e = sp.onConnect(conn); e != nil {
				logrus.Error(e)
				continue
			}
		}
		go sp.forwardConnection(conn)
	}
}

func (sp *Proxy) forwardConnection(conn net.Conn) {
	proxy, e := tls.Dial("tcp", sp.addr, &tls.Config{
		InsecureSkipVerify: true,
	})
	if e != nil {
		logrus.Error(e)
		return
	}
	logrus.Infof("successfully forwarding connection to `%s`", sp.addr)
	go io.Copy(proxy, conn)
	_, e = io.Copy(conn, proxy)
	if e != nil {
		logrus.Error(e)
		return
	}
}

func (sp *Proxy) Close() error {
	sp.isClosed = true
	return sp.lis.Close()
}
