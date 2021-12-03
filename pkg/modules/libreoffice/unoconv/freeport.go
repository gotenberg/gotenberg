package unoconv

import (
	"fmt"
	"net"
	"strconv"

	"go.uber.org/zap"
)

func freePort(logger *zap.Logger) (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("listen on the local network address: %w", err)
	}
	defer func() {
		err := listener.Close()
		if err != nil {
			logger.Error(fmt.Sprintf("close listener: %s", err.Error()))
		}
	}()

	addr := listener.Addr().String()

	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return 0, fmt.Errorf("get free port from host: %w", err)
	}

	return strconv.Atoi(portStr)
}
