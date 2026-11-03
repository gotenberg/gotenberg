package chromium

import (
	"context"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/chromedp/cdproto/network"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

func TestPrintToPdfAttrs(t *testing.T) {
	options := DefaultPdfOptions()
	options.Landscape = true
	options.PageRanges = "1-5"
	options.HeaderTemplate = "<div>secret header</div>"
	// FooterTemplate left at default, so has_footer must be false.

	got := map[string]attribute.Value{}
	for _, kv := range printToPdfAttrs(options) {
		got[string(kv.Key)] = kv.Value
		if s := kv.Value.AsString(); strings.Contains(s, "secret") || s == "1-5" {
			t.Errorf("attribute %s leaked a raw value: %q", kv.Key, s)
		}
	}

	if !got["gotenberg.chromium.print.landscape"].AsBool() {
		t.Error("expected landscape=true")
	}
	if !got["gotenberg.chromium.print.has_page_ranges"].AsBool() {
		t.Error("expected has_page_ranges=true")
	}
	if !got["gotenberg.chromium.print.has_header"].AsBool() {
		t.Error("expected has_header=true")
	}
	if got["gotenberg.chromium.print.has_footer"].AsBool() {
		t.Error("expected has_footer=false")
	}
}

func TestConversionInputAttrs(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "index.html")
	content := []byte("<html></html>")
	if err := os.WriteFile(tmp, content, 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	t.Run("file URL with non-api context", func(t *testing.T) {
		got := map[string]int64{}
		for _, kv := range conversionInputAttrs(context.Background(), "file://"+tmp) {
			got[string(kv.Key)] = kv.Value.AsInt64()
		}

		if _, ok := got["gotenberg.conversion.input.file.count"]; ok {
			t.Error("did not expect file.count for a non-api context")
		}
		if got["gotenberg.conversion.input.html.size"] != int64(len(content)) {
			t.Errorf("expected html.size=%d, got %d", len(content), got["gotenberg.conversion.input.html.size"])
		}
	})

	t.Run("remote URL yields no html.size", func(t *testing.T) {
		for _, kv := range conversionInputAttrs(context.Background(), "https://example.com") {
			if string(kv.Key) == "gotenberg.conversion.input.html.size" {
				t.Error("did not expect html.size for a remote URL")
			}
		}
	})

	t.Run("api context yields file.count", func(t *testing.T) {
		var found bool
		for _, kv := range conversionInputAttrs(&api.Context{}, "https://example.com") {
			if string(kv.Key) == "gotenberg.conversion.input.file.count" {
				found = true
				if kv.Value.AsInt64() != 0 {
					t.Errorf("expected file.count=0, got %d", kv.Value.AsInt64())
				}
			}
		}
		if !found {
			t.Error("expected file.count for an api context")
		}
	})
}

// TestRecordNetwork_Names pins the names recordNetwork records, which
// dashboards and alerts key on.
func TestRecordNetwork_Names(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	meter := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)).Meter("test")

	mod := new(Chromium)
	var err error
	mod.networkRequestsCounter, err = meter.Int64Counter("gotenberg.chromium.network.requests")
	if err != nil {
		t.Fatalf("create counter: %v", err)
	}
	mod.networkBytesCounter, err = meter.Int64Histogram("gotenberg.chromium.network.response.size")
	if err != nil {
		t.Fatalf("create histogram: %v", err)
	}

	aggregate := newNetworkAggregate()
	aggregate.onResponseReceived(&network.EventResponseReceived{
		RequestID: "1",
		Response:  &network.Response{URL: "https://cdn.example.com/logo.png?token=secret"},
	})
	aggregate.onLoadingFinished(&network.EventLoadingFinished{RequestID: "1", EncodedDataLength: 900})
	aggregate.onLoadingFailed(&network.EventLoadingFailed{RequestID: "2"})

	recorder := tracetest.NewSpanRecorder()
	tracer := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)).Tracer("test")
	ctx, span := tracer.Start(context.Background(), "chromium.pdf")
	mod.recordNetwork(ctx, span, aggregate)
	span.End()

	var rm metricdata.ResourceMetrics
	err = reader.Collect(context.Background(), &rm)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}

	outcomes := make(map[string]int64)
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "gotenberg.chromium.network.requests" {
				continue
			}
			for _, dp := range m.Data.(metricdata.Sum[int64]).DataPoints {
				if dp.Attributes.Len() != 1 {
					t.Errorf("data point attributes = %v, want only the outcome", dp.Attributes.ToSlice())
				}
				v, _ := dp.Attributes.Value("gotenberg.chromium.network.request.outcome")
				outcomes[v.AsString()] += dp.Value
			}
		}
	}
	if outcomes["ok"] != 1 || outcomes["failed"] != 1 || len(outcomes) != 2 {
		t.Errorf("gotenberg.chromium.network.requests by outcome = %v, want one ok and one failed", outcomes)
	}

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(ended))
	}

	attrs := make(map[string]int64)
	for _, kv := range ended[0].Attributes() {
		attrs[string(kv.Key)] = kv.Value.AsInt64()
	}
	wantAttrs := map[string]int64{
		"gotenberg.chromium.network.request.count":        2,
		"gotenberg.chromium.network.request.failed.count": 1,
		"gotenberg.chromium.network.response.size":        900,
		"gotenberg.chromium.network.origin.count":         1,
	}
	if !maps.Equal(attrs, wantAttrs) {
		t.Errorf("span attributes = %v, want %v", attrs, wantAttrs)
	}

	events := ended[0].Events()
	if len(events) != 1 || events[0].Name != "gotenberg.chromium.heaviest_resource" {
		t.Fatalf("span events = %v, want one gotenberg.chromium.heaviest_resource", events)
	}
	eventAttrs := make(map[string]string)
	for _, kv := range events[0].Attributes {
		eventAttrs[string(kv.Key)] = kv.Value.Emit()
	}
	wantEventAttrs := map[string]string{
		"url.full": "https://cdn.example.com/logo.png",
		"gotenberg.chromium.heaviest_resource.size": "900",
	}
	if !maps.Equal(eventAttrs, wantEventAttrs) {
		t.Errorf("event attributes = %v, want %v", eventAttrs, wantEventAttrs)
	}
}

// TestChromium_SpanNames pins the names of the conversion spans, which
// dashboards and alerts key on.
func TestChromium_SpanNames(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))
	t.Cleanup(func() { otel.SetTracerProvider(previous) })

	meter := noop.NewMeterProvider().Meter("test")
	mod := &Chromium{
		args: browserArguments{binPath: "/nonexistent/chromium"},
		supervisor: &gotenberg.ProcessSupervisorMock{
			RunMock: func(context.Context, *slog.Logger, func() error) error { return nil },
		},
	}
	mod.networkBytesCounter, _ = meter.Int64Histogram("gotenberg.chromium.network.response.size")

	outputPath := filepath.Join(t.TempDir(), "out")
	logger := slog.New(slog.DiscardHandler)

	err := mod.Pdf(context.Background(), logger, "https://example.com", outputPath, DefaultPdfOptions())
	if err != nil {
		t.Fatalf("pdf: %v", err)
	}
	err = mod.Screenshot(context.Background(), logger, "https://example.com", outputPath, DefaultScreenshotOptions())
	if err != nil {
		t.Fatalf("screenshot: %v", err)
	}

	var names []string
	for _, span := range recorder.Ended() {
		names = append(names, span.Name())
	}
	if want := []string{"chromium.pdf", "chromium.screenshot"}; !slices.Equal(names, want) {
		t.Errorf("span names = %v, want %v", names, want)
	}
}
