package proxy

import (
	"crypto/tls"
	"net"
	"testing"

	utls "github.com/refraction-networking/utls"
)

func TestNewUtlsConn_Fingerprints(t *testing.T) {
	fps := []string{
		"chrome", "firefox", "ios", "android", "edge", "safari", "360", "qq", "random", "unknown",
	}

	for _, fp := range fps {
		opts := &Options{
			TlsFingerprint: fp,
			SslInsecure:    true,
		}
		clientHello := &tls.ClientHelloInfo{
			ServerName:        "example.com",
			SupportedProtos:   []string{"h2", "http/1.1"},
			CipherSuites:      []uint16{tls.TLS_AES_128_GCM_SHA256},
			SupportedVersions: []uint16{tls.VersionTLS13},
		}
		
		conn := &net.TCPConn{} // Dummy connection
		// We expect NewUtlsConn not to panic, but it might fail or return valid UClient struct.
		// Since we pass a dummy conn, doing Handshake will fail, but creation should succeed.
		
		uConn, err := NewUtlsConn(conn, opts, clientHello)
		if err != nil {
			t.Errorf("NewUtlsConn failed for %s: %v", fp, err)
		}
		if uConn == nil {
			t.Errorf("NewUtlsConn returned nil for %s", fp)
		}
	}
}

func TestNewUtlsConn_ClientMirror(t *testing.T) {
	opts := &Options{
		TlsFingerprint: "client",
		SslInsecure:    true,
	}
	clientHello := &tls.ClientHelloInfo{
		ServerName:        "example.com",
		SupportedProtos:   []string{"h2"},
		CipherSuites:      []uint16{0x1301},
		SupportedVersions: []uint16{tls.VersionTLS13},
		SupportedCurves:   []tls.CurveID{tls.X25519},
		SupportedPoints:   []uint8{0},
	}
	conn := &net.TCPConn{}
	uConn, err := NewUtlsConn(conn, opts, clientHello)
	if err != nil {
		t.Fatalf("NewUtlsConn failed for client mirror: %v", err)
	}
	if uConn == nil {
		t.Fatal("NewUtlsConn returned nil")
	}
}

func TestEnsureSNI(t *testing.T) {
	spec := &utls.ClientHelloSpec{
		Extensions: []utls.TLSExtension{},
	}
	ensureSNI(spec, "example.com")
	
	hasSNI := false
	for _, ext := range spec.Extensions {
		if sni, ok := ext.(*utls.SNIExtension); ok {
			hasSNI = true
			if sni.ServerName != "example.com" {
				t.Errorf("Expected SNI example.com, got %s", sni.ServerName)
			}
		}
	}
	if !hasSNI {
		t.Error("SNI extension not added")
	}

	// Test existing SNI
	spec = &utls.ClientHelloSpec{
		Extensions: []utls.TLSExtension{
			&utls.SNIExtension{ServerName: "existing.com"},
		},
	}
	ensureSNI(spec, "new.com")
	// Should NOT add new SNI
	count := 0
	for _, ext := range spec.Extensions {
		if _, ok := ext.(*utls.SNIExtension); ok {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Expected 1 SNI extension, got %d", count)
	}
}

func TestUtlsStateToTlsState(t *testing.T) {
	uState := utls.ConnectionState{
		Version:            tls.VersionTLS13,
		HandshakeComplete:  true,
		DidResume:          false,
		CipherSuite:        tls.TLS_AES_128_GCM_SHA256,
		NegotiatedProtocol: "h2",
		ServerName:         "example.com",
	}
	
	tlsState := UtlsStateToTlsState(uState)
	
	if tlsState.Version != uState.Version {
		t.Error("Version mismatch")
	}
	if tlsState.ServerName != uState.ServerName {
		t.Error("ServerName mismatch")
	}
	if tlsState.NegotiatedProtocol != uState.NegotiatedProtocol {
		t.Error("NextProto mismatch")
	}
}
