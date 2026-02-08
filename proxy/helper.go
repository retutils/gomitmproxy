package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

var normalErrMsgs []string = []string{
	"read: connection reset by peer",
	"write: broken pipe",
	"i/o timeout",
	"net/http: TLS handshake timeout",
	"io: read/write on closed pipe",
	"connect: connection refused",
	"connect: connection reset by peer",
	"use of closed network connection",
}

// 仅打印预料之外的错误信息
func logErr(log *log.Entry, err error) (loged bool) {
	msg := err.Error()

	for _, str := range normalErrMsgs {
		if strings.Contains(msg, str) {
			log.Debug(err)
			return
		}
	}

	log.Error(err)
	loged = true
	return
}

// 转发流量
func transfer(log *log.Entry, server, client io.ReadWriteCloser) {
	done := make(chan struct{})
	defer close(done)

	errChan := make(chan error)
	go func() {
		// client -> server
		_, err := io.Copy(server, client)
		log.Debugln("client copy end", err)
		// Close server write side? net.Conn usually fully closes.
		// If server is *net.TCPConn we could CloseWrite.
		// But here it's io.ReadWriteCloser.
		// server.Close() closes both ways usually.
		
		// If we close server here, the other goroutine reading from server might get error or EOF.
		// But other goroutine copies server -> client.
		
		server.Close() 
		
		select {
		case <-done:
			return
		case errChan <- err:
			return
		}
	}()
	go func() {
		// server -> client
		_, err := io.Copy(client, server)
		log.Debugln("server copy end", err)
		
		// If we close client, the other goroutine reading from client (if not finished) gets EOF/error.
		client.Close()

		if clientConn, ok := client.(*wrapClientConn); ok {
			if tcpConn, ok := clientConn.Conn.(*net.TCPConn); ok {
				// CloseRead? client is already Closed above.
				// CloseRead is for half-close when we want to continue writing?
				// But we just finished writing to client (Copy(client, server)).
				// So we are done with client.
				_ = tcpConn
			}
		}
		
		select {
		case <-done:
			return
		case errChan <- err:
			return
		}
	}()

	// Wait for 2? No, if one side closes, we should probably stop?
	// But io.Copy returns when EOF.
	// If one direction finishes, we might want to wait for other or close other.
	// Original code waited for 2.
	// But if one errors, it returns.
	
	// Issue in test: using pipes, io.Copy might block if not closed properly.
	// We ensure pipes are closed in test.
	
	for i := 0; i < 2; i++ {
		select {
		case err := <-errChan:
			if err != nil {
				logErr(log, err)
				return 
			}
		case <-done:
			return
		case <-time.After(30 * time.Second):
			return
		}
	}
}

func httpError(w http.ResponseWriter, error string, code int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Proxy-Authenticate", `Basic realm="proxy"`) // Indicates that the proxy server requires client credentials
	w.WriteHeader(code)
	fmt.Fprintln(w, error)
}
