package main

import (
	"github.com/retutils/gomitmproxy/proxy"
	log "github.com/sirupsen/logrus"
)

type RewriteHost struct {
	proxy.BaseAddon
}

func (a *RewriteHost) ClientConnected(client *proxy.ClientConn) {
	// necessary
	client.UpstreamCert = false
}

func (a *RewriteHost) Requestheaders(f *proxy.Flow) {
	log.Printf("Host: %v, Method: %v, Scheme: %v", f.Request.URL.Host, f.Request.Method, f.Request.URL.Scheme)
	f.Request.URL.Host = "www.baidu.com"
	f.Request.URL.Scheme = "http"
	log.Printf("After: %v", f.Request.URL)
}

func main() {
	if err := Run(); err != nil {
		log.Fatal(err)
	}
}

func Run() error {
	opts := &proxy.Options{
		Addr:              ":9080",
		StreamLargeBodies: 1024 * 1024 * 5,
	}

	p, err := proxy.NewProxy(opts)
	if err != nil {
		return err
	}

	p.AddAddon(&RewriteHost{})
	p.AddAddon(&proxy.LogAddon{})

	return p.Start()
}
