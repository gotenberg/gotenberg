package api

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strconv"
)

func freePort(logger *slog.Logger) (int, error) {
	netListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("listen on the local network address: %w", err)
	}
	defer func() {
		err := netListener.Close()
		if err != nil {
			logger.ErrorContext(context.Background(), fmt.Sprintf("close network listener: %s", err.Error()))
		}
	}()

	addr := netListener.Addr().String()

	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return 0, fmt.Errorf("get free port from host: %w", err)
	}

	return strconv.Atoi(portStr)
}
