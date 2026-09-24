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
// See https://opentelemetry.io/.
package otel
