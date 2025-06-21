package netio

import (
	"net"
	"context"
)

// Generic function that write all data over TCP/UDP
func writeAll(writeFunc func([]byte) (int, error), data []byte) error {
	written := 0	
	stop := len(data)

	for written < stop {
		n, err := writeFunc(data[written:])

		if err != nil {
			return err
		}

		written += n
	}

	return nil
}

// Write all the data over TCP
func WriteTCP(conn net.Conn, data []byte) error {
	return writeAll(conn.Write, data)
}

// Write all the data over UDP
func WriteUDP(conn *net.UDPConn, data []byte) error {
	return writeAll(conn.Write, data)
}

// Write all the data to the given UDP addr
func WriteUDPAddr(conn *net.UDPConn, addr *net.UDPAddr, data []byte) error {
	writeFunc := func(b []byte) (int, error) {
		return conn.WriteToUDP(b, addr)
	}

	return writeAll(writeFunc, data)
}

// Channelize the read opearation of a TCP connection
func TCPReadAsChannel(
	ctx context.Context,
	conn net.Conn, 
	sendCh chan <- []byte,
) error {
	buf := make([]byte, 4096)
	defer close(sendCh)

	for {
		select {
		case <-ctx.Done():
			return nil
		default: 
			// Read from the TCP connection
			n, err := conn.Read(buf)

			// Stop upon error or EOF occurs
			if err != nil || n == 0{ 
				sendCh <- []byte{}
				return err 
			}

			// Send data, which will block. Allow cancellation happen
			data := []byte{}
			sendCh <- append(data, buf[:n]...)
		}
	}
}
