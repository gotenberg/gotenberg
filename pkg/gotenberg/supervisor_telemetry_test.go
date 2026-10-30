package gotenberg

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func newTestMetricReader(t *testing.T) *sdkmetric.ManualReader {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	previous := otel.GetMeterProvider()
	otel.SetMeterProvider(provider)
	t.Cleanup(func() { otel.SetMeterProvider(previous) })
	return reader
}

// metricPoints returns, per attribute value of key, the sum of a counter or
// the count of a histogram named name.
func metricPoints(t *testing.T, reader *sdkmetric.ManualReader, name string, key attribute.Key) map[string]int64 {
	t.Helper()

	var rm metricdata.ResourceMetrics
	err := reader.Collect(context.Background(), &rm)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}

	points := make(map[string]int64)
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			switch data := m.Data.(type) {
			case metricdata.Sum[int64]:
				for _, dp := range data.DataPoints {
					v, _ := dp.Attributes.Value(key)
					points[v.AsString()] += dp.Value
				}
			case metricdata.Histogram[float64]:
				for _, dp := range data.DataPoints {
					v, _ := dp.Attributes.Value(key)
					points[v.AsString()] += int64(dp.Count)
				}
			}
		}
	}

	return points
}

func TestProcessSupervisor_Run_Metrics(t *testing.T) {
	errBoom := errors.New("boom")

	for _, tc := range []struct {
		scenario       string
		maxQueueSize   int64
		queued         int64
		taskErr        error
		expectStatus   string
		expectReason   string
		expectDuration bool
	}{
		{scenario: "success", expectStatus: "success", expectDuration: true},
		{scenario: "error", taskErr: errBoom, expectStatus: "error", expectReason: "boom_reason", expectDuration: true},
		{scenario: "timeout", taskErr: context.DeadlineExceeded, expectStatus: "timeout", expectReason: "timeout", expectDuration: true},
		{scenario: "canceled", taskErr: context.Canceled, expectStatus: "canceled", expectReason: "context_cancelled", expectDuration: true},
		{
			// The run never starts, but its wait is still recorded.
			scenario:     "queue full",
			maxQueueSize: 1,
			queued:       1,
			expectStatus: "error",
			expectReason: "queue_size_exceeded",
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			reader := newTestMetricReader(t)

			process := &ProcessMock{
				StartMock:   func(context.Context, *slog.Logger) error { return nil },
				StopMock:    func(*slog.Logger) error { return nil },
				HealthyMock: func(*slog.Logger) bool { return true },
			}
			ps, err := NewProcessSupervisor(slog.New(slog.DiscardHandler), "test", process, SupervisorOptions{
				MaxQueueSize: tc.maxQueueSize,
				ErrorType: func(err error) string {
					if errors.Is(err, errBoom) {
						return "boom_reason"
					}
					return ClassifyError(err)
				},
			})
			if err != nil {
				t.Fatalf("create supervisor: %v", err)
			}
			ps.(*processSupervisor).reqQueueSize.Store(tc.queued)

			_ = ps.Run(context.Background(), slog.New(slog.DiscardHandler), func() error { return tc.taskErr })

			if got := metricPoints(t, reader, "test.requests.total", "status"); got[tc.expectStatus] != 1 || len(got) != 1 {
				t.Errorf("test.requests.total = %v, want one %q", got, tc.expectStatus)
			}

			errs := metricPoints(t, reader, "test.errors.total", "reason")
			if tc.expectReason == "" && len(errs) != 0 {
				t.Errorf("test.errors.total = %v, want none", errs)
			}
			if tc.expectReason != "" && (errs[tc.expectReason] != 1 || len(errs) != 1) {
				t.Errorf("test.errors.total = %v, want one %q", errs, tc.expectReason)
			}

			if got := metricPoints(t, reader, "test.queue.wait.duration", "status"); got[tc.expectStatus] != 1 {
				t.Errorf("test.queue.wait.duration = %v, want one %q", got, tc.expectStatus)
			}

			durations := metricPoints(t, reader, "test.conversion.duration", "status")
			if tc.expectDuration != (durations[tc.expectStatus] == 1) {
				t.Errorf("test.conversion.duration = %v, want recorded: %t", durations, tc.expectDuration)
			}
		})
	}
}

func TestProcessSupervisor_Run_AnnotatesCallerSpan(t *testing.T) {
	recorder := newTestSpanRecorder(t)

	process := &ProcessMock{
		StartMock:   func(context.Context, *slog.Logger) error { return nil },
		StopMock:    func(*slog.Logger) error { return nil },
		HealthyMock: func(*slog.Logger) bool { return true },
	}
	ps := newTestSupervisor(t, slog.New(slog.DiscardHandler), process, 0, 0, 1, 0)

	ctx, span := Tracer().Start(context.Background(), "caller")
	err := ps.Run(ctx, slog.New(slog.DiscardHandler), func() error { return nil })
	span.End()
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	caller := findSpan(recorder, "caller")
	if caller == nil {
		t.Fatal("caller span not recorded")
	}

	found := make(map[attribute.Key]bool)
	for _, kv := range caller.Attributes() {
		found[kv.Key] = true
	}
	for _, key := range []attribute.Key{"gotenberg.queue.depth_at_arrival", "gotenberg.conversions_since_last_restart"} {
		if !found[key] {
			t.Errorf("caller span misses %q", key)
		}
	}
}
