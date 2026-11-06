package main

import (
	"bytes"
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
		return status, data[i+4:], nil
	}
	return status, data, nil
}
