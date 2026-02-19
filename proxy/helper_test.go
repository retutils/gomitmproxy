package proxy

import (
	"bytes"
	"errors"
	"io"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestTransfer_Error(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	
	// Mock connections that fail on read/write
	server := &mockConn{readErr: errors.New("server read error")}
	client := &mockConn{}
	
	// Should log error and return
	transfer(log, server, client)
}

func TestTransfer_WrapClientConn(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	
	server := &mockConn{}
    p, _ := NewProxy(&Options{Addr: ":0"})
    
    mConn := &mockConn{}
	client := newWrapClientConn(mConn, p)
    client.connCtx = &ConnContext{
        ClientConn: &ClientConn{
            Conn: mConn,
        },
    }
	
	transfer(log, server, client)
}

func TestLogErr(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	logrus.SetOutput(&buf)
	defer logrus.SetOutput(os.Stderr)
	log := logrus.WithField("test", "logErr")

	// 1. Normal errors (should return false and log debug)
	logrus.SetLevel(logrus.DebugLevel)
	for _, msg := range normalErrMsgs {
		buf.Reset()
		err := errors.New("some prefix " + msg + " some suffix")
		// logErr returns "loged bool" -> true if logged as error?
		// "return true means unusual error logged"
		// "return false/implicit means handled as normal debug"
		if logErr(log, err) {
			// Actually normalErrMsgs are logged as Debug and return is implicit false?
			// Let's check implementation
			// if contains normal msg -> log.Debug(err); return
			// else log.Error(err); return true
			// So if it returns true, it's NOT a normal error.
			t.Errorf("expected logErr to return false (handled as normal) for msg: %s", msg)
		}
		if buf.Len() == 0 {
			// Debug logging should happen
			t.Errorf("expected debug log for normal error: %s", msg)
		}
	}

	// 2. Unexpected errors (should return true and log error)
	logrus.SetLevel(logrus.InfoLevel)
	buf.Reset()
	err := errors.New("unexpected error")
	if !logErr(log, err) {
		t.Error("expected logErr to return true for unexpected error")
	}
	if buf.Len() == 0 {
		t.Error("expected error log for unexpected error")
	}
}

func TestHttpError(t *testing.T) {
	w := httptest.NewRecorder()
	httpError(w, "auth required", 407)
	
	resp := w.Result()
	if resp.StatusCode != 407 {
		t.Errorf("want status 407, got %d", resp.StatusCode)
	}
	if resp.Header.Get("Proxy-Authenticate") == "" {
		t.Error("want Proxy-Authenticate header")
	}
	body, _ := io.ReadAll(resp.Body)
	if string(bytes.TrimSpace(body)) != "auth required" {
		t.Errorf("want body 'auth required', got %q", string(body))
	}
}

func TestTransfer(t *testing.T) {
	t.Skip("Skipping complex transfer test for now, focusing on logErr and httpError")
}
