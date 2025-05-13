package transport

import (
	"fmt"
	"net"
)

// writeAll is a helper function that handles the common write-all pattern
func writeAll(writeFunc func([]byte) (int, error), data []byte) error {
	written := 0
	stop := len(data)

	for written < stop {
		n, err := writeFunc(data[written:])
		if err != nil {
			return fmt.Errorf("failed to write data: %w", err)
		}
		written += n
	}

	return nil
}

func WriteAllUDP(conn *net.UDPConn, data []byte) error {
	return writeAll(conn.Write, data)
}

func WriteAllUDPAddr(conn *net.UDPConn, addr *net.UDPAddr, data []byte) error {
	writeFunc := func(b []byte) (int, error) {
		return conn.WriteToUDP(b, addr)
	}
	return writeAll(writeFunc, data)
}

func WriteAllTCP(conn net.Conn, data []byte) error {
	return writeAll(conn.Write, data)
}
