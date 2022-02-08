package uno

import (
	"context"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestListener_start(t *testing.T) {
	tests := []struct {
		name           string
		listener       listener
		expectStartErr bool
	}{
		{
			name:     "nominal behavior",
			listener: newLibreOfficeListener(zap.NewNop(), os.Getenv("LIBREOFFICE_BIN_PATH"), time.Duration(10)*time.Second, 10),
		},
		{
			name:           "non-exit code 81 on first start",
			listener:       newLibreOfficeListener(zap.NewNop(), "foo", time.Duration(10)*time.Second, 10),
			expectStartErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.listener.start(zap.NewNop())

			if tc.expectStartErr && err == nil {
				t.Fatalf("expected listener.start() error, but got none")
			}

			if !tc.expectStartErr && err != nil {
				t.Fatalf("expected no error from listener.start(), but got: %v", err)
			}

			if err != nil {
				if tc.listener.healthy() {
					t.Error("expected a non-running LibreOffice listener")
				}

				return
			}

			err = tc.listener.stop(zap.NewNop())
			if err != nil {
				t.Fatalf("expected no error from listener.stop(), but got: %v", err)
			}
		})
	}
}

func TestListener_stop(t *testing.T) {
	listener := newLibreOfficeListener(
		zap.NewNop(),
		os.Getenv("LIBREOFFICE_BIN_PATH"),
		time.Duration(10)*time.Second,
		10,
	)

	err := listener.start(zap.NewNop())
	if err != nil {
		t.Fatalf("expected no error from listener.start(), but got: %v", err)
	}

	err = listener.stop(zap.NewNop())
	if err != nil {
		t.Errorf("expected no error from listener.stop(), but got: %v", err)
	}
}

func TestListener_restart(t *testing.T) {
	listener := newLibreOfficeListener(
		zap.NewNop(),
		os.Getenv("LIBREOFFICE_BIN_PATH"),
		time.Duration(10)*time.Second,
		10,
	)

	err := listener.start(zap.NewNop())
	if err != nil {
		t.Fatalf("expected no error from listener.start(), but got: %v", err)
	}

	err = listener.restart(zap.NewNop())
	if err != nil {
		t.Errorf("expected no error from listener.stop(), but got: %v", err)
	}

	if !listener.healthy() {
		t.Error("expected an healthy LibreOffice listener")
	}
}

func TestListener_lock(t *testing.T) {
	tests := []struct {
		name          string
		listener      listener
		ctx           context.Context
		teardown      func(listener listener) error
		expectLockErr bool
	}{
		{
			name: "nominal behavior",
			listener: func() listener {
				listener := newLibreOfficeListener(zap.NewNop(), os.Getenv("LIBREOFFICE_BIN_PATH"), time.Duration(10)*time.Second, 10)

				err := listener.start(zap.NewNop())
				if err != nil {
					t.Fatalf("expected no error from listener.start(), but got: %v", err)
				}

				return listener
			}(),
			ctx: context.Background(),
			teardown: func(listener listener) error {
				return listener.stop(zap.NewNop())
			},
		},
		{
			name: "unhealthy listener",
			listener: func() listener {
				listener := newLibreOfficeListener(zap.NewNop(), os.Getenv("LIBREOFFICE_BIN_PATH"), time.Duration(10)*time.Second, 10)

				err := listener.start(zap.NewNop())
				if err != nil {
					t.Fatalf("expected no error from listener.start(), but got: %v", err)
				}

				err = listener.stop(zap.NewNop())
				if err != nil {
					t.Fatalf("expected no error from listener.stop(), but got: %v", err)
				}

				return listener
			}(),
			ctx: context.Background(),
			teardown: func(listener listener) error {
				return listener.stop(zap.NewNop())
			},
		},
		{
			name: "context done",
			listener: func() listener {
				listener := newLibreOfficeListener(zap.NewNop(), os.Getenv("LIBREOFFICE_BIN_PATH"), time.Duration(10)*time.Second, 10)

				err := listener.start(zap.NewNop())
				if err != nil {
					t.Fatalf("expected no error from listener.start(), but got: %v", err)
				}

				err = listener.lock(context.Background(), zap.NewNop())
				if err != nil {
					t.Fatalf("expected no error from listener.lock(), but got: %v", err)
				}

				return listener
			}(),
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return ctx
			}(),
			expectLockErr: true,
			teardown: func(listener listener) error {
				return listener.stop(zap.NewNop())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				err := tc.teardown(tc.listener)
				if err != nil {
					t.Errorf("expected no error from tc.teardown(), but got: %v", err)
				}
			}()

			err := tc.listener.lock(tc.ctx, zap.NewNop())

			if tc.expectLockErr && err == nil {
				t.Fatalf("expected listener.lock() error, but got none")
			}

			if !tc.expectLockErr && err != nil {
				t.Fatalf("expected no error from listener.lock(), but got: %v", err)
			}
		})
	}
}

func TestListener_unlock(t *testing.T) {
	tests := []struct {
		name     string
		listener listener
		teardown func(listener listener) error
	}{
		{
			name: "nominal behavior",
			listener: func() listener {
				listener := newLibreOfficeListener(zap.NewNop(), os.Getenv("LIBREOFFICE_BIN_PATH"), time.Duration(10)*time.Second, 10)

				err := listener.start(zap.NewNop())
				if err != nil {
					t.Fatalf("expected no error from listener.start(), but got: %v", err)
				}

				err = listener.lock(context.Background(), zap.NewNop())
				if err != nil {
					t.Fatalf("expected no error from listener.lock(), but got: %v", err)
				}

				return listener
			}(),
			teardown: func(listener listener) error {
				return listener.stop(zap.NewNop())
			},
		},
		{
			name: "unhealthy listener",
			listener: func() listener {
				listener := newLibreOfficeListener(zap.NewNop(), os.Getenv("LIBREOFFICE_BIN_PATH"), time.Duration(10)*time.Second, 10)

				err := listener.start(zap.NewNop())
				if err != nil {
					t.Fatalf("expected no error from listener.start(), but got: %v", err)
				}

				err = listener.lock(context.Background(), zap.NewNop())
				if err != nil {
					t.Fatalf("expected no error from listener.lock(), but got: %v", err)
				}

				err = listener.stop(zap.NewNop())
				if err != nil {
					t.Fatalf("expected no error from listener.stop(), but got: %v", err)
				}

				return listener
			}(),
			teardown: func(listener listener) error {
				return listener.stop(zap.NewNop())
			},
		},
		{
			name: "threshold reached",
			listener: func() listener {
				listener := newLibreOfficeListener(zap.NewNop(), os.Getenv("LIBREOFFICE_BIN_PATH"), time.Duration(10)*time.Second, 1)

				err := listener.start(zap.NewNop())
				if err != nil {
					t.Fatalf("expected no error from listener.start(), but got: %v", err)
				}

				err = listener.lock(context.Background(), zap.NewNop())
				if err != nil {
					t.Fatalf("expected no error from listener.lock(), but got: %v", err)
				}

				return listener
			}(),
			teardown: func(listener listener) error {
				return listener.stop(zap.NewNop())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				err := tc.teardown(tc.listener)
				if err != nil {
					t.Errorf("expected no error from tc.teardown(), but got: %v", err)
				}
			}()

			err := tc.listener.unlock(zap.NewNop())
			if err != nil {
				t.Errorf("expected no error from listener.unlock(), but got: %v", err)
			}
		})
	}
}

func TestListener_port(t *testing.T) {
	listener := newLibreOfficeListener(
		zap.NewNop(),
		os.Getenv("LIBREOFFICE_BIN_PATH"),
		time.Duration(10)*time.Second,
		10,
	)

	err := listener.start(zap.NewNop())
	if err != nil {
		t.Fatalf("expected no error from listener.start(), but got: %v", err)
	}

	port := listener.port()
	if port == 0 {
		t.Error("expected a non-zero value from listener.port")
	}

	err = listener.stop(zap.NewNop())
	if err != nil {
		t.Errorf("expected no error from listener.stop(), but got: %v", err)
	}
}

func TestListener_queue(t *testing.T) {
	listener := newLibreOfficeListener(
		zap.NewNop(),
		os.Getenv("LIBREOFFICE_BIN_PATH"),
		time.Duration(10)*time.Second,
		10,
	)

	err := listener.start(zap.NewNop())
	if err != nil {
		t.Fatalf("expected no error from listener.start(), but got: %v", err)
	}

	defer func() {
		err := listener.stop(zap.NewNop())
		if err != nil {
			t.Errorf("expected no error from listener.stop(), but got: %v", err)
		}
	}()

	queueLength := listener.queue()
	if queueLength != 0 {
		t.Fatalf("expected a zero value from listener.queue(), but got %d", queueLength)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10)*time.Second)

	err = listener.lock(ctx, zap.NewNop())
	if err != nil {
		t.Fatalf("expected no error from listener.lock(), but got: %v", err)
	}

	queueLength = listener.queue()
	if queueLength != 0 {
		t.Fatalf("expected a zero value from listener.queue(), but got %d", queueLength)
	}

	go func() {
		_ = listener.lock(ctx, zap.NewNop())
	}()

	time.Sleep(time.Duration(100) * time.Millisecond)

	queueLength = listener.queue()
	if queueLength != 1 {
		t.Fatalf("expected 1 from listener.queue(), but got %d", queueLength)
	}

	go func() {
		_ = listener.lock(ctx, zap.NewNop())
	}()

	time.Sleep(time.Duration(100) * time.Millisecond)

	queueLength = listener.queue()
	if queueLength != 2 {
		t.Fatalf("expected 2 from listener.queue(), but got %d", queueLength)
	}

	cancel()

	time.Sleep(time.Duration(100) * time.Millisecond)

	queueLength = listener.queue()
	if queueLength != 0 {
		t.Fatalf("expected a zero value from listener.queue(), but got %d", queueLength)
	}
}

func TestListener_healthy(t *testing.T) {
	listener := &libreOfficeListener{
		binPath:      os.Getenv("LIBREOFFICE_BIN_PATH"),
		startTimeout: time.Duration(10) * time.Second,
		threshold:    10,
		lockChan:     make(chan struct{}, 1),
		logger:       zap.NewNop(),
	}

	err := listener.start(zap.NewNop())
	if err != nil {
		t.Fatalf("expected no error from listener.start(), but got: %v", err)
	}

	if !listener.healthy() {
		t.Error("expected an healthy LibreOffice listener")
	}

	err = listener.stop(zap.NewNop())
	if err != nil {
		t.Fatalf("expected no error from listener.stop(), but got: %v", err)
	}

	if listener.healthy() {
		t.Errorf("expected a non-healthy LibreOffice listener")
	}

	listener.restarting = true

	if !listener.healthy() {
		t.Error("expected an healthy LibreOffice listener")
	}
}
