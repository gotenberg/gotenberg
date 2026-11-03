package gotenberg

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

// runMetrics are the gotenberg.<engine>.* instruments a [ProcessSupervisor]
// records.
type runMetrics struct {
	requests  metric.Int64Counter
	queueWait metric.Float64Histogram
	duration  metric.Float64Histogram
}

// newRunMetrics creates the run metrics of s, and the gauges observing its
// state.
func newRunMetrics(s *processSupervisor) (runMetrics, error) {
	meter := Meter()
	name := func(suffix string) string { return "gotenberg." + s.engine + "." + suffix }

	gauges := []struct {
		suffix, description, unit string
		observe                   func() int64
	}{
		{"request.active", "Current number of active %s requests", "{request}", s.ActiveTasksCount},
		{"queue.size", "Current number of %s requests waiting to be treated", "{request}", s.ReqQueueSize},
	}
	for _, g := range gauges {
		_, err := meter.Int64ObservableGauge(
			name(g.suffix),
			metric.WithDescription(fmt.Sprintf(g.description, s.engine)),
			metric.WithUnit(g.unit),
			metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
				o.Observe(g.observe())
				return nil
			}),
		)
		if err != nil {
			return runMetrics{}, fmt.Errorf("create %s gauge: %w", name(g.suffix), err)
		}
	}

	_, err := meter.Int64ObservableCounter(
		name("process.restarts"),
		metric.WithDescription(fmt.Sprintf("Current number of %s restarts", s.engine)),
		metric.WithUnit("{restart}"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			o.Observe(s.RestartsCount())
			return nil
		}),
	)
	if err != nil {
		return runMetrics{}, fmt.Errorf("create %s counter: %w", name("process.restarts"), err)
	}

	var m runMetrics

	m.requests, err = meter.Int64Counter(
		name("requests"),
		metric.WithDescription(fmt.Sprintf("Total number of %s requests", s.engine)),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return runMetrics{}, fmt.Errorf("create %s counter: %w", name("requests"), err)
	}

	durationBuckets := metric.WithExplicitBucketBoundaries(0.5, 1, 2, 5, 10, 30, 60)

	m.duration, err = meter.Float64Histogram(
		name("conversion.duration"),
		metric.WithDescription(fmt.Sprintf("Duration of %s runs", s.engine)),
		metric.WithUnit("s"),
		durationBuckets,
	)
	if err != nil {
		return runMetrics{}, fmt.Errorf("create %s histogram: %w", name("conversion.duration"), err)
	}

	m.queueWait, err = meter.Float64Histogram(
		name("queue.wait.duration"),
		metric.WithDescription(fmt.Sprintf("Duration of waiting in queue for %s runs", s.engine)),
		metric.WithUnit("s"),
		durationBuckets,
	)
	if err != nil {
		return runMetrics{}, fmt.Errorf("create %s histogram: %w", name("queue.wait.duration"), err)
	}

	return m, nil
}

// runMeasure follows one run, from its arrival to its outcome.
type runMeasure struct {
	s         *processSupervisor
	arrival   time.Time
	taskStart time.Time
}

// measureRun annotates the span in ctx with the state of s on arrival, and
// starts measuring the run.
func (s *processSupervisor) measureRun(ctx context.Context) *runMeasure {
	trace.SpanFromContext(ctx).SetAttributes(
		attribute.Int64("gotenberg.queue.depth_at_arrival", s.ReqQueueSize()),
		attribute.Int64("gotenberg.conversions_since_last_restart", s.ConversionsSinceRestart()),
	)

	return &runMeasure{s: s, arrival: time.Now()}
}

// taskStarted marks the end of the queue wait.
func (m *runMeasure) taskStarted() {
	m.taskStart = time.Now()
}

// end records the outcome of the run. A run rejected before its task started
// records its whole duration as queue wait.
func (m *runMeasure) end(ctx context.Context, err error) {
	end := time.Now()

	// Per the semconv, a failure carries an error.type and a success no
	// attribute, so the failed runs are the requests with an error.type. It
	// is the same value as on the span of the run, so that a metric data
	// point and a span of one failure share it.
	var attrs []attribute.KeyValue
	if err != nil {
		attrs = append(attrs, semconv.ErrorTypeKey.String(m.s.errorType(err)))
	}
	outcome := metric.WithAttributes(attrs...)

	m.s.metrics.requests.Add(ctx, 1, outcome)

	if m.taskStart.IsZero() {
		m.s.metrics.queueWait.Record(ctx, end.Sub(m.arrival).Seconds(), outcome)
		return
	}

	m.s.metrics.queueWait.Record(ctx, m.taskStart.Sub(m.arrival).Seconds(), outcome)
	m.s.metrics.duration.Record(ctx, end.Sub(m.taskStart).Seconds(), outcome)
}
