package gotenberg

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"
)

type appStub struct {
	name string
	stop func(ctx context.Context) error
}

func (a *appStub) Start() error                   { return nil }
func (a *appStub) StartupMessage() string         { return "" }
func (a *appStub) Stop(ctx context.Context) error { return a.stop(ctx) }

type ingressStub struct{ appStub }

func (i *ingressStub) Ingress() {}

func TestStopApps(t *testing.T) {
	for _, tc := range []struct {
		scenario      string
		ingressErr    error
		expectErr     bool
		expectStopped []string
	}{
		{
			scenario:      "the ingress drains before the engines stop",
			expectStopped: []string{"api", "chromium", "libreoffice"},
		},
		{
			scenario:      "the engines stop even when the ingress fails",
			ingressErr:    context.DeadlineExceeded,
			expectErr:     true,
			expectStopped: []string{"chromium", "libreoffice"},
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			var mu sync.Mutex
			var events []string
			record := func(event string) {
				mu.Lock()
				defer mu.Unlock()
				events = append(events, event)
			}

			engine := func(name string) *appStub {
				return &appStub{name: name, stop: func(context.Context) error {
					record(name + " stop")
					return nil
				}}
			}
			ingress := &ingressStub{appStub{name: "api", stop: func(context.Context) error {
				// Leave the engines time to stop early, if they were to.
				time.Sleep(50 * time.Millisecond)
				record("api drained")
				return tc.ingressErr
			}}}

			// The ingress comes last, so only the grouping orders it first.
			apps := []App{engine("chromium"), engine("libreoffice"), ingress}

			var stopped []string
			err := StopApps(context.Background(), apps, func(app App) {
				mu.Lock()
				defer mu.Unlock()
				switch a := app.(type) {
				case *appStub:
					stopped = append(stopped, a.name)
				case *ingressStub:
					stopped = append(stopped, a.name)
				}
			})

			if tc.expectErr != (err != nil) {
				t.Fatalf("expected error %t, got %v", tc.expectErr, err)
			}
			if tc.ingressErr != nil && !errors.Is(err, tc.ingressErr) {
				t.Errorf("expected the error to wrap %v, got %v", tc.ingressErr, err)
			}

			if len(events) != 3 || events[0] != "api drained" {
				t.Errorf("expected the API to drain before any engine stops, got %v", events)
			}

			slices.Sort(stopped)
			if !slices.Equal(stopped, tc.expectStopped) {
				t.Errorf("stopped = %v, want %v", stopped, tc.expectStopped)
			}
		})
	}
}
