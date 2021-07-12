package squid

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

type TunnelAuth struct {
	Username string
	Password string
}

type TunnelConfig struct {
	Auth       *TunnelAuth
	ProxyAddr  string
	RemoteAddr string
	BindAddr   string
	SSH        bool
	CertPath   string
}

type Tunnel struct {
	lis       net.Listener
	configs   *TunnelConfig
	port      uint16
	isClosed  bool
	onConnect func(net.Conn) error
}

func NewTunnel(config *TunnelConfig) (*Tunnel, error) {
	if len(config.ProxyAddr) == 0 {
		return nil, errors.New("please provide the address of the proxy server")
	}
	if len(config.RemoteAddr) == 0 {
		return nil, errors.New("please provide the address of the remote server")
	}
	return &Tunnel{
		configs: config,
	}, nil
}

func (st *Tunnel) OnClientConnect(cb func(net.Conn) error) {
	if cb != nil {
		st.onConnect = cb
	}
}

func (st *Tunnel) Start(port uint16) (e error) {
	st.port = port
	if st.port == 0 {
		st.port, e = GetFreePort()
		if e != nil {
			return e
		}
	}
	st.lis, e = net.Listen("tcp", fmt.Sprintf("%s:%d", LOCALHOST, st.port))
	if e != nil {
		return e
	}
	go st.accept()
	return nil
}

func (st *Tunnel) accept() {
	for {
		conn, e := st.lis.Accept()
		if e != nil {
			if st.isClosed {
				return
			}
			logrus.Error(e)
			continue
		}
		if st.onConnect != nil {
			if e = st.onConnect(conn); e != nil {
				logrus.Error(e)
				continue
			}
		}
		if st.configs.SSH {
			go st.forwardSSH(conn)
		} else {
			go st.forwardConnection(conn)
		}
	}
}

func (st *Tunnel) dialCoordinatorViaCONNECT(proxy *url.URL) (net.Conn, error) {
	proxyAddr := proxy.Host
	if proxy.Port() == "" {
		proxyAddr = net.JoinHostPort(proxyAddr, "80")
	}
	remoteAddr := st.configs.RemoteAddr
	remoteInfo := strings.Split(remoteAddr, ":")
	if len(remoteInfo) == 1 && st.configs.SSH {
		remoteAddr += ":22"
	}
	var d net.Dialer
	toCtx, toCnl := context.WithTimeout(context.Background(), 30*time.Second)
	defer toCnl()
	c, err := d.DialContext(toCtx, "tcp", proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("dialing proxy %q failed: %v", proxyAddr, err)
	}
	basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", st.configs.Auth.Username, st.configs.Auth.Password)))
	fmt.Fprintf(c, "CONNECT %s HTTP/1.1\r\nHost: %s\r\nProxy-Authorization: %s\r\n\r\n", remoteAddr, proxy.Hostname(), basicAuth)
	br := bufio.NewReader(c)
	res, err := http.ReadResponse(br, nil)
	if err != nil {
		return nil, fmt.Errorf("reading HTTP response from CONNECT to %s via proxy %s failed: %v",
			proxy.String(), proxyAddr, err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("proxy error from %s while dialing %s: %v", proxyAddr, proxy.String(), res.Status)
	}
	if br.Buffered() > 0 {
		return nil, fmt.Errorf("unexpected %d bytes of buffered data from CONNECT proxy %q",
			br.Buffered(), proxyAddr)
	}
	return c, nil
}

func (st *Tunnel) forwardSSH(conn net.Conn) {
	defer conn.Close()
	h, e := url.Parse(fmt.Sprintf("http://%s", st.configs.ProxyAddr))
	if e != nil {
		logrus.Error(e)
		return
	}
	pconn, e := st.dialCoordinatorViaCONNECT(h)
	if e != nil {
		logrus.Error(e)
		return
	}
	pemKey, e := st.getPemKey(st.configs.CertPath)
	if e != nil {
		logrus.Error(e)
		return
	}
	sshConfig := &ssh.ClientConfig{
		User: "kratos",
		Auth: []ssh.AuthMethod{
			pemKey,
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	sshConn, sshChan, sshReq, e := ssh.NewClientConn(pconn, st.configs.RemoteAddr, sshConfig)
	if e != nil {
		logrus.Error(e)
		return
	}

	sshClient := ssh.NewClient(sshConn, sshChan, sshReq)

	bindConn, e := sshClient.Dial("tcp", st.configs.BindAddr)
	if e != nil {
		logrus.Error(e)
		return
	}
	defer bindConn.Close()

	go io.Copy(bindConn, conn)
	_, e = io.Copy(conn, bindConn)
	if e != nil {
		logrus.Error(e)
		return
	}
}

func (st *Tunnel) forwardConnection(conn net.Conn) {
	defer conn.Close()
	h, e := url.Parse(fmt.Sprintf("http://%s", st.configs.ProxyAddr))
	if e != nil {
		logrus.Error(e)
		return
	}
	pconn, e := st.dialCoordinatorViaCONNECT(h)
	if e != nil {
		logrus.Error(e)
		return
	}
	go io.Copy(pconn, conn)
	_, e = io.Copy(conn, pconn)
	if e != nil {
		logrus.Error(e)
		return
	}
}

func (st *Tunnel) GetPort() uint16 {
	return st.port
}

func (st *Tunnel) Close() error {
	st.isClosed = true
	return st.lis.Close()
}

func (st *Tunnel) getPemKey(path string) (ssh.AuthMethod, error) {
	key, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	signer, e := ssh.ParsePrivateKey(key)
	if e != nil {
		return nil, e
	}
	return ssh.PublicKeys(signer), e
}
