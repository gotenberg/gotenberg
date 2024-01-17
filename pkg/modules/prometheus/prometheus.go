package prometheus

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	flag "github.com/spf13/pflag"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

func init() {
	gotenberg.MustRegisterModule(new(Prometheus))
}

// Prometheus is a module which collects metrics and exposes them via an HTTP
// route.
type Prometheus struct {
	namespace           string
	interval            time.Duration
	disableRouteLogging bool
	disableCollect      bool

	metrics  []gotenberg.Metric
	registry *prometheus.Registry
}

// Descriptor returns a [Prometheus]'s module descriptor.
func (mod *Prometheus) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID: "prometheus",
		FlagSet: func() *flag.FlagSet {
			fs := flag.NewFlagSet("prometheus", flag.ExitOnError)
			fs.String("prometheus-namespace", "gotenberg", "Set the namespace of modules' metrics")
			fs.Duration("prometheus-collect-interval", time.Duration(1)*time.Second, "Set the interval for collecting modules' metrics")
			fs.Bool("prometheus-disable-route-logging", false, "Disable the route logging")
			fs.Bool("prometheus-disable-collect", false, "Disable the collect of metrics")

			return fs
		}(),
		New: func() gotenberg.Module { return new(Prometheus) },
	}
}

// Provision sets the modules properties.
func (mod *Prometheus) Provision(ctx *gotenberg.Context) error {
	flags := ctx.ParsedFlags()
	mod.namespace = flags.MustString("prometheus-namespace")
	mod.interval = flags.MustDuration("prometheus-collect-interval")
	mod.disableRouteLogging = flags.MustBool("prometheus-disable-route-logging")
	mod.disableCollect = flags.MustBool("prometheus-disable-collect")

	if mod.disableCollect {
		// Exit early.
		return nil
	}

	// Get metrics from modules.
	mods, err := ctx.Modules(new(gotenberg.MetricsProvider))
	if err != nil {
		return fmt.Errorf("get metrics providers: %w", err)
	}

	metricsProviders := make([]gotenberg.MetricsProvider, len(mods))
	for i, metricsProvider := range mods {
		metricsProviders[i] = metricsProvider.(gotenberg.MetricsProvider)
	}

	for _, metricsProvider := range metricsProviders {
		metrics, err := metricsProvider.Metrics()
		if err != nil {
			return fmt.Errorf("get metrics: %w", err)
		}

		mod.metrics = append(mod.metrics, metrics...)
	}

	mod.registry = prometheus.NewRegistry()

	return nil
}

// Validate validates the module properties.
func (mod *Prometheus) Validate() error {
	if mod.disableCollect {
		// Exit early.
		return nil
	}

	if mod.namespace == "" {
		return errors.New("namespace must not be empty")
	}

	metricsMap := make(map[string]string, len(mod.metrics))

	for _, metric := range mod.metrics {
		if metric.Name == "" {
			return errors.New("metric name cannot be empty")
		}

		if metric.Read == nil {
			return fmt.Errorf("metric '%s' has nil read method", metric.Name)
		}

		if _, ok := metricsMap[metric.Name]; ok {
			return fmt.Errorf("metric '%s' is already registered", metric.Name)
		}

		metricsMap[metric.Name] = metric.Name
	}

	return nil
}

// Start starts the collect.
func (mod *Prometheus) Start() error {
	if mod.disableCollect {
		// Exit early.
		return nil
	}

	for _, metric := range mod.metrics {
		gauge := prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: mod.namespace,
				Name:      metric.Name,
				Help:      metric.Description,
			},
		)

		mod.registry.MustRegister(gauge)

		go func(gauge prometheus.Gauge, metric gotenberg.Metric) {
			for {
				gauge.Set(metric.Read())
				time.Sleep(mod.interval)
			}
		}(gauge, metric)
	}

	return nil
}

// StartupMessage returns a custom startup message.
func (mod *Prometheus) StartupMessage() string {
	if mod.disableCollect {
		return "collect disabled"
	}

	return "collecting metrics"
}

// Stop does nothing.
func (mod *Prometheus) Stop(ctx context.Context) error {
	return nil
}

// Routes returns the HTTP route.
func (mod *Prometheus) Routes() ([]api.Route, error) {
	if mod.disableCollect {
		return nil, nil
	}

	return []api.Route{
		{
			Method:         http.MethodGet,
			Path:           "/prometheus/metrics",
			DisableLogging: mod.disableRouteLogging,
			Handler: echo.WrapHandler(
				promhttp.HandlerFor(mod.registry, promhttp.HandlerOpts{}),
			),
		},
	}, nil
}

// Interface guards.
var (
	_ gotenberg.Module      = (*Prometheus)(nil)
	_ gotenberg.Provisioner = (*Prometheus)(nil)
	_ gotenberg.Validator   = (*Prometheus)(nil)
	_ gotenberg.App         = (*Prometheus)(nil)
	_ api.Router            = (*Prometheus)(nil)
)
