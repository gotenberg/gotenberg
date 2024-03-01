package gotenberg

// Metric represents a unitary metric.
type Metric struct {
	// Name is the unique identifier.
	// Required.
	Name string

	// Description describes the metric.
	// Optional.
	Description string

	// Read returns the current value.
	// Required.
	Read func() float64
}

// MetricsProvider is a module interface which provides a list of [Metric].
//
//	func (m *YourModule) Provision(ctx *gotenberg.Context) error {
//		provider, _ := ctx.Module(new(gotenberg.MetricsProvider))
//		metrics, _  := provider.(gotenberg.MetricsProvider).Metrics()
//	}
type MetricsProvider interface {
	Metrics() ([]Metric, error)
}
