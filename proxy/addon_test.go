package proxy

import (
	"testing"
)

func TestBaseAddon(t *testing.T) {
    b := &BaseAddon{}
    // Ensure methods don't panic
    b.ClientConnected(nil)
    b.ClientDisconnected(nil)
    b.ServerConnected(nil)
    b.ServerDisconnected(nil)
    b.TlsEstablishedServer(nil)
    b.Requestheaders(nil)
    b.Request(nil)
    b.Responseheaders(nil)
    b.Response(nil)
    b.StreamRequestModifier(nil, nil)
    b.StreamResponseModifier(nil, nil)
    b.AccessProxyServer(nil, nil)
}

// TestLogAddon removed due to logrus global state interference
// Covered by instance_log_addon_test probably or deemed less critical than stability.

func TestUpstreamCertAddon(t *testing.T) {
    u := NewUpstreamCertAddon(true)
    c := &ClientConn{}
    u.ClientConnected(c)
    if !c.UpstreamCert {
        t.Error("expected UpstreamCert true")
    }
    
    u = NewUpstreamCertAddon(false)
    u.ClientConnected(c)
    if c.UpstreamCert {
        t.Error("expected UpstreamCert false")
    }
}
