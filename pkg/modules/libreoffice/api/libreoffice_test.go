package api

import (
	"log/slog"
	"net"
	"testing"
	"time"
)

func TestDialUno(t *testing.T) {
	// The first bytes LibreOffice 25.x sends on a new URP connection.
	greeting := []byte("\x00\x00\x00e\x00\x00\x00\x01\xf8\x04\x96\x00\x00'com.sun.star.bridge.XProtocolProperties\x15UrpProtocolProperties")

	for _, tc := range []struct {
		scenario  string
		serve     func(conn net.Conn)
		listen    bool
		expectErr bool
	}{
		{
			scenario: "LibreOffice greeting",
			serve: func(conn net.Conn) {
				_, _ = conn.Write(greeting)
			},
			listen: true,
		},
		{
			scenario: "greeting split across writes",
			serve: func(conn net.Conn) {
				_, _ = conn.Write(greeting[:20])
				time.Sleep(10 * time.Millisecond)
				_, _ = conn.Write(greeting[20:])
			},
			listen: true,
		},
		{
			// An HTTP server, such as the LibreOffice outbound proxy, waits
			// for the request and sends nothing.
			// See https://github.com/gotenberg/gotenberg/issues/1713.
			scenario:  "silent listener",
			serve:     func(conn net.Conn) {},
			listen:    true,
			expectErr: true,
		},
		{
			scenario: "listener that sends something else",
			serve: func(conn net.Conn) {
				_, _ = conn.Write([]byte("SSH-2.0-OpenSSH_9.9\r\n"))
			},
			listen:    true,
			expectErr: true,
		},
		{
			scenario: "listener that streams without the greeting",
			serve: func(conn net.Conn) {
				for range 64 {
					_, err := conn.Write(make([]byte, 64))
					if err != nil {
						return
					}
				}
			},
			listen:    true,
			expectErr: true,
		},
		{
			scenario:  "nothing listening",
			expectErr: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatalf("listen: %v", err)
			}
			port := listener.Addr().(*net.TCPAddr).Port

			if !tc.listen {
				err = listener.Close()
				if err != nil {
					t.Fatalf("close listener: %v", err)
				}
			} else {
				done := make(chan struct{})
				t.Cleanup(func() {
					close(done)
					_ = listener.Close()
				})

				go func() {
					conn, err := listener.Accept()
					if err != nil {
						return
					}
					defer conn.Close()

					tc.serve(conn)
					<-done
				}()
			}

			err = dialUno(slog.New(slog.DiscardHandler), port, 200*time.Millisecond)

			if !tc.expectErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}

			if tc.expectErr && err == nil {
				t.Error("expected an error but got none")
			}
		})
	}
}
