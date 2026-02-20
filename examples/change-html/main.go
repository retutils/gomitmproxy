package main

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/retutils/gomitmproxy/proxy"
	log "github.com/sirupsen/logrus"
)

var titleRegexp = regexp.MustCompile("(<title>)(.*?)(</title>)")

type ChangeHtml struct {
	proxy.BaseAddon
}

func (c *ChangeHtml) Response(f *proxy.Flow) {
	contentType := f.Response.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		return
	}

	// change html <title> end with: " - go-mitmproxy"
	f.Response.ReplaceToDecodedBody()
	f.Response.Body = titleRegexp.ReplaceAll(f.Response.Body, []byte("${1}${2} - go-mitmproxy${3}"))
	f.Response.Header.Set("Content-Length", strconv.Itoa(len(f.Response.Body)))
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

	p.AddAddon(&ChangeHtml{})

	return p.Start()
}
