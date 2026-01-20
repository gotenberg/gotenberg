package prometheus

import (
	"net/http"
	"slices"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	flag "github.com/spf13/pflag"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

func init() {
	gotenberg.MustRegisterModule(new(Prometheus))
}

// Prometheus is a module that collects metrics and exposes them via an HTTP
// route.
type Prometheus struct {
	metricsPath         string
	disableRouteLogging bool
	disable             bool
}

// Descriptor returns a [Prometheus]'s module descriptor.
func (mod *Prometheus) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID: "prometheus",
		FlagSet: func() *flag.FlagSet {
			fs := flag.NewFlagSet("prometheus", flag.ExitOnError)
			fs.String("prometheus-metrics-path", "/prometheus/metrics", "Path for Prometheus metrics endpoint")
			fs.Bool("prometheus-disable-route-logging", false, "Disable the route logging")

			// Deprecated flags.
			fs.String("prometheus-namespace", "gotenberg", "Set the namespace of modules' metrics")
			fs.Duration("prometheus-collect-interval", time.Duration(5)*time.Second, "Set the interval for collecting modules' metrics")
			fs.Bool("prometheus-disable-collect", false, "Disable the collect of metrics")

			return fs
		}(),
		New: func() gotenberg.Module { return new(Prometheus) },
	}
}

// Provision sets the module properties.
func (mod *Prometheus) Provision(ctx *gotenberg.Context) error {
	flags := ctx.ParsedFlags()
	mod.metricsPath = flags.MustString("prometheus-metrics-path")
	mod.disableRouteLogging = flags.MustBool("prometheus-disable-route-logging")

	protocols := flags.MustStringSlice("telemetry-metric-exporter-protocols")
	mod.disable = !slices.Contains(protocols, gotenberg.PrometheusTelemetryMetricExporterProtocol)

	return nil
}

// Routes returns the HTTP route.
func (mod *Prometheus) Routes() ([]api.Route, error) {
	if mod.disable {
		return nil, nil
	}

	return []api.Route{
		{
			Method:         http.MethodGet,
			Path:           mod.metricsPath,
			DisableLogging: mod.disableRouteLogging,
			Handler: echo.WrapHandler(
				promhttp.HandlerFor(gotenberg.PrometheusRegistry(), promhttp.HandlerOpts{}),
			),
		},
	}, nil
}

// Interface guards.
var (
	_ gotenberg.Module      = (*Prometheus)(nil)
	_ gotenberg.Provisioner = (*Prometheus)(nil)
	_ api.Router            = (*Prometheus)(nil)
)
