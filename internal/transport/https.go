package transport

import (
	"fmt"
	"bufio"
	"net"
	"net/http"
)

func ParseHTTPConnectHost(conn net.Conn) (string, error) {
	req, err := http.ReadRequest(bufio.NewReader(conn))

	// Reading HTTP request error
	if err != nil {
		return "", fmt.Errorf("can't reading HTTP request. %s\n", err)
	}

	// Only allow HTTP CONNECT method
	if req.Method != "CONNECT" {
		return "", fmt.Errorf("error HTTP method, not CONNECT allowed")
	}

	return req.Host, nil
}

// Notify client tunnel is established
func NotifyClientOnSuccess(conn net.Conn) error {
    _, err := conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

    if err != nil {
		return fmt.Errorf("can't notify HTTPS client on success. %s\n", err)
    }

    return nil
}

// Notify client tunnel can't be established
func NotifyClientOnFailure(conn net.Conn) error {
    _, err := conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))

    if err != nil {
		return fmt.Errorf("can't notify HTTPS client on failure. %s\n", err)
    }

    return nil
}