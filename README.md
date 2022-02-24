# Squid

A simple TCP traffic forwarding package to help connecting to instances or hosts behind firewalls

## Usage

```go
  package main

  import "github.com/melordy/squid"

  ProxyURL := "example.proxy.com:443"
  proxy, e := squid.NewProxy(ProxyURL)
  if e != nil {
    logrus.Fatal(e)
  }
  proxy.Start(0)
  defer proxy.Close()
  logrus.WithField("PORT", proxy.GetPort()).Info("Proxy server listening...")

  tunnel, e := squid.NewTunnel(&squid.TunnelConfig{
    Auth: &squid.TunnelAuth{
      Username: *username,
      Password: *password,
    },
    ProxyAddr:  fmt.Sprintf("%s:%d", squid.LOCALHOST, proxy.GetPort()),
    RemoteAddr: *serviceHost,
    BindAddr:   *bindService,
    SSH:        *isSSH,
    CertPath:   *pemKey,
  })
  if e != nil {
    logrus.Fatal(e)
  }
  tunnel.Start(*tunnelPort)
  defer tunnel.Close()
  logrus.WithField("PORT", tunnel.GetPort()).Info("Tunnel server listening...")

  graceful := make(chan os.Signal, 1)
  signal.Notify(graceful, os.Interrupt)
  <-graceful
  logrus.Info("Thank you for using squid tool, have a great day!")
```

## Disclaimer

This package is only tested within my own requirements,
it might not work for your need, please use with caution.

## Author

- Kevin Xu
