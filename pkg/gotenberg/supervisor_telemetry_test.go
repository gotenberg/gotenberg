package gotenberg

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"slices"
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
// the count of a histogram named name. A data point without key counts under
// "".
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
		expectType     string
		expectDuration bool
	}{
		// A success carries no error.type.
		{scenario: "success", expectDuration: true},
		{scenario: "error", taskErr: errBoom, expectType: "boom_reason", expectDuration: true},
		{scenario: "timeout", taskErr: context.DeadlineExceeded, expectType: "timeout", expectDuration: true},
		{scenario: "canceled", taskErr: context.Canceled, expectType: "context_cancelled", expectDuration: true},
		{
			// The run never starts, but its wait is still recorded.
			scenario:     "queue full",
			maxQueueSize: 1,
			queued:       1,
			expectType:   "queue_size_exceeded",
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

			if got := metricPoints(t, reader, "gotenberg.test.requests", "error.type"); got[tc.expectType] != 1 || len(got) != 1 {
				t.Errorf("gotenberg.test.requests = %v, want one %q", got, tc.expectType)
			}

			if got := metricPoints(t, reader, "gotenberg.test.queue.wait.duration", "error.type"); got[tc.expectType] != 1 || len(got) != 1 {
				t.Errorf("gotenberg.test.queue.wait.duration = %v, want one %q", got, tc.expectType)
			}

			durations := metricPoints(t, reader, "gotenberg.test.conversion.duration", "error.type")
			if tc.expectDuration != (durations[tc.expectType] == 1) {
				t.Errorf("gotenberg.test.conversion.duration = %v, want recorded: %t", durations, tc.expectDuration)
			}
		})
	}
}

// TestProcessSupervisor_MetricNames pins the instrument names and their
// attribute keys, which dashboards and alerts key on. The Prometheus exporter
// derives its names from them.
func TestProcessSupervisor_MetricNames(t *testing.T) {
	reader := newTestMetricReader(t)

	process := &ProcessMock{
		StartMock:   func(context.Context, *slog.Logger) error { return nil },
		StopMock:    func(*slog.Logger) error { return nil },
		HealthyMock: func(*slog.Logger) bool { return true },
	}
	ps, err := NewProcessSupervisor(slog.New(slog.DiscardHandler), "test", process, SupervisorOptions{})
	if err != nil {
		t.Fatalf("create supervisor: %v", err)
	}

	// A failed run feeds every instrument with its error.type, and a
	// successful one feeds them without it.
	_ = ps.Run(context.Background(), slog.New(slog.DiscardHandler), func() error { return errors.New("boom") })
	_ = ps.Run(context.Background(), slog.New(slog.DiscardHandler), func() error { return nil })

	var rm metricdata.ResourceMetrics
	err = reader.Collect(context.Background(), &rm)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}

	got := make(map[string][]attribute.Key)
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			got[m.Name] = metricAttributeKeys(t, m)
		}
	}

	want := map[string][]attribute.Key{
		"gotenberg.test.request.active":      {},
		"gotenberg.test.queue.size":          {},
		"gotenberg.test.process.restarts":    {},
		"gotenberg.test.requests":            {"error.type"},
		"gotenberg.test.conversion.duration": {"error.type"},
		"gotenberg.test.queue.wait.duration": {"error.type"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("recorded metrics and attribute keys = %v, want %v", got, want)
	}
}

// metricAttributeKeys returns the sorted attribute keys of the data points of
// m.
func metricAttributeKeys(t *testing.T, m metricdata.Metrics) []attribute.Key {
	t.Helper()

	var sets []attribute.Set
	switch data := m.Data.(type) {
	case metricdata.Gauge[int64]:
		for _, dp := range data.DataPoints {
			sets = append(sets, dp.Attributes)
		}
	case metricdata.Sum[int64]:
		for _, dp := range data.DataPoints {
			sets = append(sets, dp.Attributes)
		}
	case metricdata.Histogram[float64]:
		for _, dp := range data.DataPoints {
			sets = append(sets, dp.Attributes)
		}
	default:
		t.Fatalf("metric %q has unexpected data %T", m.Name, m.Data)
	}

	keys := make(map[attribute.Key]bool)
	for _, set := range sets {
		for _, kv := range set.ToSlice() {
			keys[kv.Key] = true
		}
	}

	sorted := make([]attribute.Key, 0, len(keys))
	for key := range keys {
		sorted = append(sorted, key)
	}
	slices.Sort(sorted)

	return sorted
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
