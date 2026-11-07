package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"
)

// rawHTTP is a minimal HTTP/1.1 client over a socket — replaces net/http
// for loopback probes and saves ~1.5 MB of TLS+client stack from the binary.
func rawHTTP(addr, method, path, body string, timeout time.Duration) (int, []byte, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return 0, nil, err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(timeout))

	var req strings.Builder
	fmt.Fprintf(&req, "%s %s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n", method, path, addr)
	if body != "" {
		fmt.Fprintf(&req, "Content-Type: application/json\r\nContent-Length: %d\r\n", len(body))
	}
	req.WriteString("\r\n")
	req.WriteString(body)
	if _, err := conn.Write([]byte(req.String())); err != nil {
		return 0, nil, err
	}

	buf := make([]byte, 8192)
	var data []byte
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil || len(data) > 65536 {
			break
		}
	}
	if len(data) == 0 {
		return 0, nil, fmt.Errorf("empty response")
	}
	status := 0
	if i := bytes.IndexByte(data, '\n'); i > 8 {
		line := strings.TrimSpace(string(data[:i]))
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			fmt.Sscanf(parts[1], "%d", &status)
		}
	}
	if i := bytes.Index(data, []byte("\r\n\r\n")); i >= 0 {
		head := bytes.ToLower(data[:i])
		body := data[i+4:]
		if bytes.Contains(head, []byte("transfer-encoding:")) &&
			bytes.Contains(head, []byte("chunked")) {
			body = dechunk(body)
		}
		return status, body, nil
	}
	return status, data, nil
}

// httpsGet is the TLS twin of rawHTTP — minimal HTTPS/1.1 for the one
// place fabi leaves the box: checking its own release version on
// GitHub (user-invoked only, never autonomous).
func httpsGet(host, path string, timeout time.Duration) (int, []byte, error) {
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: timeout}, "tcp", host+":443",
		&tls.Config{ServerName: host})
	if err != nil {
		return 0, nil, err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(timeout))
	var req strings.Builder
	fmt.Fprintf(&req,
		"GET %s HTTP/1.1\r\nHost: %s\r\nUser-Agent: fabi\r\nConnection: close\r\n\r\n",
		path, host)
	if _, err := conn.Write([]byte(req.String())); err != nil {
		return 0, nil, err
	}
	buf := make([]byte, 8192)
	var data []byte
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil || len(data) > 1<<20 {
			break
		}
	}
	if len(data) == 0 {
		return 0, nil, fmt.Errorf("empty response")
	}
	status := 0
	if i := bytes.IndexByte(data, '\n'); i > 8 {
		line := strings.TrimSpace(string(data[:i]))
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			fmt.Sscanf(parts[1], "%d", &status)
		}
	}
	if i := bytes.Index(data, []byte("\r\n\r\n")); i >= 0 {
		head := bytes.ToLower(data[:i])
		body := data[i+4:]
		if bytes.Contains(head, []byte("transfer-encoding:")) &&
			bytes.Contains(head, []byte("chunked")) {
			body = dechunk(body)
		}
		return status, body, nil
	}
	return status, data, nil
}

// dechunk decodes HTTP/1.1 chunked transfer-encoding.
func dechunk(b []byte) []byte {
	var out []byte
	for len(b) > 0 {
		i := bytes.IndexByte(b, '\n')
		if i <= 0 {
			break
		}
		var n int64
		if _, err := fmt.Sscanf(strings.TrimSpace(string(b[:i])), "%x", &n); err != nil || n <= 0 {
			break
		}
		b = b[i+1:]
		if int(n) > len(b) {
			n = int64(len(b))
		}
		out = append(out, b[:n]...)
		b = b[n:]
	}
	return out
}
