package scenario

import (
	"fmt"
	"net"
	"strconv"
)

type hostPort struct {
	port     int
	listener net.Listener
}

func (h *hostPort) use() int {
	defer h.listener.Close()
	return h.port
}

func (h *hostPort) close() {
	h.listener.Close()
}

func freePort() (*hostPort, error) {
	netListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen on the local network address: %w", err)
	}

	addr := netListener.Addr().String()

	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		defer netListener.Close()
		return nil, fmt.Errorf("get free port from host: %w", err)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		defer netListener.Close()
		return nil, fmt.Errorf("convert port %q to int: %w", portStr, err)
	}

	return &hostPort{
		port:     port,
		listener: netListener,
	}, nil
}
