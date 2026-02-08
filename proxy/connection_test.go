package proxy

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func testGetResponse(t *testing.T, endpoint string, client *http.Client) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest("GET", endpoint, nil)
	handleError(t, err)
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	handleError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	handleError(t, err)
	return resp, body
}

type testConnectionAddon struct {
	BaseAddon
}

func (a *testConnectionAddon) Response(f *Flow) {
	tlsStr := "0"
	if f.ConnContext.ClientConn.Tls {
		tlsStr = "1"
	}
	f.Response.Header.Add("tls", tlsStr)

	pStr := "null"
	if f.ConnContext.ClientConn.NegotiatedProtocol != "" {
		pStr = f.ConnContext.ClientConn.NegotiatedProtocol
	}
	f.Response.Header.Add("protocol", pStr)
}

func TestConnection(t *testing.T) {
	helper := &testProxyHelper{
		server:    &http.Server{},
		proxyAddr: ":29087",
	}
	helper.init(t)
	helper.server.TLSConfig.NextProtos = []string{"h2"}
	httpEndpoint := helper.httpEndpoint
	httpsEndpoint := helper.httpsEndpoint
	testProxy := helper.testProxy
	testProxy.AddAddon(&testConnectionAddon{})
	getProxyClient := helper.getProxyClient
	defer helper.ln.Close()
	go helper.server.Serve(helper.ln)
	defer helper.tlsPlainLn.Close()
	go helper.server.ServeTLS(helper.tlsPlainLn, "", "")
	go testProxy.Start()
	time.Sleep(time.Millisecond * 10) // wait for test proxy startup

	t.Run("ClientConn state", func(t *testing.T) {
		t.Run("http", func(t *testing.T) {
			client := getProxyClient()
			resp, _ := testGetResponse(t, httpEndpoint, client)
			if resp.Header.Get("tls") != "0" {
				t.Fatalf("expected %s, but got %s", "0", resp.Header.Get("tls"))
			}
			if resp.Header.Get("protocol") != "null" {
				t.Fatalf("expected %s, but got %s", "null", resp.Header.Get("protocol"))
			}
		})

		t.Run("https", func(t *testing.T) {
			client := getProxyClient()
			resp, _ := testGetResponse(t, httpsEndpoint, client)
			if resp.Header.Get("tls") != "1" {
				t.Fatalf("expected %s, but got %s", "1", resp.Header.Get("tls"))
			}
			if resp.Header.Get("protocol") != "null" {
				t.Fatalf("expected %s, but got %s", "null", resp.Header.Get("protocol"))
			}
		})

		t.Run("h2", func(t *testing.T) {
			client := &http.Client{
				Transport: &http.Transport{
					ForceAttemptHTTP2: true,
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
					Proxy: func(r *http.Request) (*url.URL, error) {
						return url.Parse("http://127.0.0.1" + helper.proxyAddr)
					},
				},
			}
			resp, _ := testGetResponse(t, httpsEndpoint, client)
			if resp.Header.Get("tls") != "1" {
				t.Fatalf("expected %s, but got %s", "1", resp.Header.Get("tls"))
			}
			if resp.Header.Get("protocol") != "h2" {
				t.Fatalf("expected %s, but got %s", "h2", resp.Header.Get("protocol"))
			}
		})
	})
}

func TestConnectionOffUpstreamCert(t *testing.T) {
	helper := &testProxyHelper{
		server:    &http.Server{},
		proxyAddr: ":29088",
	}
	helper.init(t)
	helper.server.TLSConfig.NextProtos = []string{"h2"}
	httpEndpoint := helper.httpEndpoint
	httpsEndpoint := helper.httpsEndpoint
	testProxy := helper.testProxy
	testProxy.AddAddon(NewUpstreamCertAddon(false))
	testProxy.AddAddon(&testConnectionAddon{})
	getProxyClient := helper.getProxyClient
	defer helper.ln.Close()
	go helper.server.Serve(helper.ln)
	defer helper.tlsPlainLn.Close()
	go helper.server.ServeTLS(helper.tlsPlainLn, "", "")
	go testProxy.Start()
	time.Sleep(time.Millisecond * 10) // wait for test proxy startup

	t.Run("ClientConn state", func(t *testing.T) {
		t.Run("http", func(t *testing.T) {
			client := getProxyClient()
			resp, _ := testGetResponse(t, httpEndpoint, client)
			if resp.Header.Get("tls") != "0" {
				t.Fatalf("expected %s, but got %s", "0", resp.Header.Get("tls"))
			}
			if resp.Header.Get("protocol") != "null" {
				t.Fatalf("expected %s, but got %s", "null", resp.Header.Get("protocol"))
			}
		})

		t.Run("https", func(t *testing.T) {
			client := getProxyClient()
			resp, _ := testGetResponse(t, httpsEndpoint, client)
			if resp.Header.Get("tls") != "1" {
				t.Fatalf("expected %s, but got %s", "1", resp.Header.Get("tls"))
			}
			if resp.Header.Get("protocol") != "null" {
				t.Fatalf("expected %s, but got %s", "null", resp.Header.Get("protocol"))
			}
		})

		t.Run("h2 not support", func(t *testing.T) {
			client := &http.Client{
				Transport: &http.Transport{
					ForceAttemptHTTP2: true,
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
					Proxy: func(r *http.Request) (*url.URL, error) {
						return url.Parse("http://127.0.0.1" + helper.proxyAddr)
					},
				},
			}
			resp, _ := testGetResponse(t, httpsEndpoint, client)
			if resp.Header.Get("tls") != "1" {
				t.Fatalf("expected %s, but got %s", "1", resp.Header.Get("tls"))
			}
			if resp.Header.Get("protocol") != "http/1.1" {
				t.Fatalf("expected %s, but got %s", "h2", resp.Header.Get("protocol"))
			}
		})
	})

}

func TestConnection_JSON(t *testing.T) {
	c := newClientConn(&mockConn{})
	b, err := c.MarshalJSON()
	if err != nil {
		t.Errorf("ClientConn MarshalJSON failed: %v", err)
	}
	if len(b) == 0 {
		t.Error("ClientConn JSON empty")
	}

	s := newServerConn()
	s.Conn = &mockConn{}
	b, err = s.MarshalJSON()
	if err != nil {
		t.Errorf("ServerConn MarshalJSON failed: %v", err)
	}
	if len(b) == 0 {
		t.Error("ServerConn JSON empty")
	}
	
	s.Conn = nil
	b, _ = s.MarshalJSON()
	if len(b) == 0 {
		t.Error("ServerConn JSON empty")
	}
}

func TestConnection_Misc(t *testing.T) {
	c := newClientConn(nil)
	ctx := &ConnContext{ClientConn: c}
	if ctx.Id() != c.Id {
		t.Error("ID mismatch")
	}
	
	s := newServerConn()
	state := &tls.ConnectionState{}
	s.tlsState = state
	if s.TlsState() != state {
		t.Error("TlsState mismatch")
	}
}
