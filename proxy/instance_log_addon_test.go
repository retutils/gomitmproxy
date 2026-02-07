package proxy

import (
	"bytes"
	"net"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/atomic"
)

func TestInstanceLogAddon(t *testing.T) {
	// Setup capture
	var buf bytes.Buffer
	logrus.SetOutput(&buf)
	logrus.SetLevel(logrus.DebugLevel)
	defer func() {
		logrus.SetOutput(os.Stderr)
		logrus.SetLevel(logrus.InfoLevel)
	}()

	// Helper to create addon with captured output
	createAddon := func() *InstanceLogAddon {
		// NewInstanceLogAddonWithFile uses its own logger instance.
		// If logFilePath is empty, it uses logrus default output.
		addon := NewInstanceLogAddonWithFile(":8080", "test-instance", "")
		return addon
	}

	addon := createAddon()

	// Helper to create a dummy flow
	createFlow := func() *Flow {
		return &Flow{
			ConnContext: &ConnContext{
				ClientConn: &ClientConn{
					Conn: &dummyConn{},
				},
				ServerConn: &ServerConn{
					Conn:    &dummyConn{},
					Address: "remote:80",
				},
				FlowCount: *atomic.NewUint32(0),
			},
			Request: &Request{
				URL:    &url.URL{Scheme: "http", Host: "example.com", Path: "/"},
				Method: "GET",
				Body:   []byte("req"),
			},
			Response: &Response{
				StatusCode: 200,
				Body:       []byte("res"),
			},
		}
	}

	f := createFlow()

	// Test cases
	tests := []struct {
		name    string
		action  func()
		wantLog string
	}{
		{"ClientConnected", func() { addon.ClientConnected(f.ConnContext.ClientConn) }, "Client connected"},
		{"ClientDisconnected", func() { addon.ClientDisconnected(f.ConnContext.ClientConn) }, "Client disconnected"},
		{"ServerConnected", func() { addon.ServerConnected(f.ConnContext) }, "Server connected"},
		{"ServerDisconnected", func() { addon.ServerDisconnected(f.ConnContext) }, "Server disconnected"},
		{"TlsEstablished", func() { addon.TlsEstablishedServer(f.ConnContext) }, "TLS connection established"},
		{"Request", func() { addon.Request(f) }, "Full request received"},
		{"Response", func() { addon.Response(f) }, "Full response received"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.action()
			if !strings.Contains(buf.String(), tt.wantLog) {
				t.Errorf("want log containing %q, got %q", tt.wantLog, buf.String())
			}
		})
	}
	// Test Requestheaders (async)
	t.Run("Requestheaders", func(t *testing.T) {
		buf.Reset()
		
		// Flow needs a done channel
		f.done = make(chan struct{}) // We need access to done chan or use a constructor?
        // If Flow struct fields are public or we can use NewFlow?
        // Let's check flow.go or try to init it.
        // Assuming f is already created via struct literal in createFlow.
        // But done is likely private if lowercased?
        // LogAddon uses f.Done() which returns <-chan struct{}.
        // So we need to be able to close it.
        // If the field is private and no exported Finish/Close method, we are stuck?
        // The addon uses `<-f.Done()`. 
        // Let's assume we can use `f.Finish()` if it exists, or if `done` is public.
        // If `done` is private and unexported, we can't set it in literal `&Flow{}`.
        // usage in addon: `<-f.Done()`.
        
        // Let's try to find how Flow is initialized.
        // For now, I'll attempt to use a helper if available, or just init the channel if accessible.
        // If 'done' is lowercase, I can't set it from test package unless test is in 'proxy' package.
        // The test IS in 'dproxy' package (`package proxy`). So I can access private fields!
        
        f.done = make(chan struct{})
        
		addon.Requestheaders(f)
		
		// Check first log
		if !strings.Contains(buf.String(), "Request headers received") {
			t.Errorf("want 'Request headers received', got %q", buf.String())
		}
		
		// Signal done
		close(f.done)
		
		// Wait for async log
		// Simple retry loop
		for i := 0; i < 10; i++ {
			if strings.Contains(buf.String(), "Request completed") {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		
		if !strings.Contains(buf.String(), "Request completed") {
			t.Errorf("want 'Request completed', got %q", buf.String())
		}
	})

    t.Run("SetLogger", func(t *testing.T) {
        l := NewInstanceLoggerWithFile("addr", "name", "")
        addon.SetLogger(l)
        if addon.logger != l {
            t.Error("SetLogger failed")
        }
    })
}

type dummyConn struct {
	net.Conn
}

func (d *dummyConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1234}
}
func (d *dummyConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5678}
}
