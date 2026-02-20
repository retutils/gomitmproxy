package main

import (
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/retutils/gomitmproxy/proxy"
)

type AddHeader struct {
	proxy.BaseAddon
	count int
}

func (a *AddHeader) Responseheaders(f *proxy.Flow) {
	a.count += 1
	f.Response.Header.Add("x-count", strconv.Itoa(a.count))
}

func main() {
	if err := Run(":9080"); err != nil {
		log.Fatal(err)
	}
}

func Run(addr string) error {
	opts := &proxy.Options{
		Addr:              addr,
		StreamLargeBodies: 1024 * 1024 * 5,
	}

	p, err := proxy.NewProxy(opts)
	if err != nil {
		return err
	}

	p.AddAddon(&AddHeader{})

	return p.Start()
}
