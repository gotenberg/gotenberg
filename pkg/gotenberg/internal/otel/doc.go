// Package otel gathers initialization utilities for OpenTelemetry
// instrumentation.
//
// Significantly inspired by https://github.com/lucavallin/gotel.
//
// # Sampling
//
// No sampler is configured in code, so the SDK default applies:
// parentbased_always_on. Every trace is recorded and exported. Operators tune
// sampling through the standard environment variables, honored by the SDK:
//
//	OTEL_TRACES_SAMPLER       e.g. parentbased_traceidratio, always_off
//	OTEL_TRACES_SAMPLER_ARG   e.g. 0.1 for a 10% ratio
//
// Head sampling drops whole traces up front, including the rare slow or failed
// conversions that matter most for diagnosis. For high-throughput deployments,
// prefer keeping head sampling permissive and applying tail sampling in the
// collector (sample on error or high latency), which decides after a trace
// completes. Gotenberg emits trace-based metric exemplars, so the conversion
// histograms still link to representative traces regardless of the head
// sampling ratio.
//
// # Exporters and propagators
//
// Each signal stays off until its OTEL_TRACES_EXPORTER, OTEL_METRICS_EXPORTER
// or OTEL_LOGS_EXPORTER names an exporter: otlp, console or none, and
// prometheus for metrics. The prometheus exporter serves the metrics on its
// own listener, OTEL_EXPORTER_PROMETHEUS_HOST and OTEL_EXPORTER_PROMETHEUS_PORT,
// localhost:9464 by default. Inside a container, set the host to 0.0.0.0 and
// publish the port, or nothing outside reaches it.
//
// OTEL_PROPAGATORS selects the propagators, such as b3multi or jaeger, W3C
// Trace Context and Baggage by default. An unknown name stops the startup.
//
// See https://opentelemetry.io/.
package otel
