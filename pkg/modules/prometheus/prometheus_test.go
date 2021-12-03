package prometheus

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/prometheus/client_golang/prometheus"
)

type ProtoModule struct {
	descriptor func() gotenberg.ModuleDescriptor
}

func (mod ProtoModule) Descriptor() gotenberg.ModuleDescriptor {
	return mod.descriptor()
}

type ProtoValidator struct {
	ProtoModule
	validate func() error
}

func (mod ProtoValidator) Validate() error {
	return mod.validate()
}

type ProtoMetricsProvider struct {
	ProtoValidator
	metrics func() ([]gotenberg.Metric, error)
}

func (mod ProtoMetricsProvider) Metrics() ([]gotenberg.Metric, error) {
	return mod.metrics()
}

func TestPrometheus_Descriptor(t *testing.T) {
	descriptor := Prometheus{}.Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(Prometheus))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestPrometheus_Provision(t *testing.T) {
	for i, tc := range []struct {
		ctx           *gotenberg.Context
		expectMetrics []gotenberg.Metric
		expectErr     bool
	}{
		{
			ctx: func() *gotenberg.Context {
				fs := new(Prometheus).Descriptor().FlagSet
				err := fs.Parse([]string{"--prometheus-disable-collect=true"})

				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: fs,
					},
					nil,
				)
			}(),
		},
		{
			ctx: func() *gotenberg.Context {
				mod := struct{ ProtoMetricsProvider }{}
				mod.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.validate = func() error {
					return errors.New("foo")
				}
				mod.metrics = func() ([]gotenberg.Metric, error) {
					return nil, nil
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(Prometheus).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
			expectErr: true,
		},
		{
			ctx: func() *gotenberg.Context {
				mod := struct{ ProtoMetricsProvider }{}
				mod.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.validate = func() error {
					return nil
				}
				mod.metrics = func() ([]gotenberg.Metric, error) {
					return nil, errors.New("foo")
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(Prometheus).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
			expectErr: true,
		},
		{
			ctx: func() *gotenberg.Context {
				mod := struct{ ProtoMetricsProvider }{}
				mod.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.validate = func() error {
					return nil
				}
				mod.metrics = func() ([]gotenberg.Metric, error) {
					return []gotenberg.Metric{
						{
							Name:        "foo",
							Description: "Bar.",
						},
					}, nil
				}

				return gotenberg.NewContext(
					gotenberg.ParsedFlags{
						FlagSet: new(Prometheus).Descriptor().FlagSet,
					},
					[]gotenberg.ModuleDescriptor{
						mod.Descriptor(),
					},
				)
			}(),
			expectMetrics: []gotenberg.Metric{
				{
					Name:        "foo",
					Description: "Bar.",
				},
			},
		},
	} {
		mod := new(Prometheus)
		err := mod.Provision(tc.ctx)

		if !reflect.DeepEqual(mod.metrics, tc.expectMetrics) {
			t.Errorf("test %d: expected %+v, but got: %+v", i, tc.expectMetrics, mod.metrics)
		}

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestPrometheus_Validate(t *testing.T) {
	for i, tc := range []struct {
		namespace      string
		metrics        []gotenberg.Metric
		disableCollect bool
		expectErr      bool
	}{
		{
			disableCollect: true,
		},
		{
			namespace: "",
			expectErr: true,
		},
		{
			namespace: "foo",
			metrics: []gotenberg.Metric{
				{
					Name: "",
				},
			},
			expectErr: true,
		},
		{
			namespace: "foo",
			metrics: []gotenberg.Metric{
				{
					Name: "foo",
				},
			},
			expectErr: true,
		},
		{
			namespace: "foo",
			metrics: []gotenberg.Metric{
				{
					Name: "foo",
					Read: func() float64 {
						return 0
					},
				},
				{
					Name: "foo",
					Read: func() float64 {
						return 0
					},
				},
			},
			expectErr: true,
		},
		{
			namespace: "foo",
			metrics: []gotenberg.Metric{
				{
					Name: "foo",
					Read: func() float64 {
						return 0
					},
				},
				{
					Name: "bar",
					Read: func() float64 {
						return 0
					},
				},
			},
		},
	} {
		mod := Prometheus{
			namespace:      tc.namespace,
			metrics:        tc.metrics,
			disableCollect: tc.disableCollect,
		}

		err := mod.Validate()

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestPrometheus_Start(t *testing.T) {
	for i, tc := range []struct {
		metrics        []gotenberg.Metric
		disableCollect bool
	}{
		{
			disableCollect: true,
		},
		{
			metrics: []gotenberg.Metric{
				{
					Name: "foo",
					Read: func() float64 {
						return 0
					},
				},
			},
		},
	} {
		mod := Prometheus{
			namespace:      "foo",
			interval:       time.Duration(1) * time.Second,
			metrics:        tc.metrics,
			disableCollect: tc.disableCollect,
			registry:       prometheus.NewRegistry(),
		}

		err := mod.Start()
		if err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestPrometheus_StartupMessage(t *testing.T) {
	for i, tc := range []struct {
		disableCollect bool
		expectMessage  string
	}{
		{
			disableCollect: true,
			expectMessage:  "collect disabled",
		},
		{
			expectMessage: "collecting metrics",
		},
	} {
		mod := Prometheus{
			disableCollect: tc.disableCollect,
		}

		actual := mod.StartupMessage()
		if actual != tc.expectMessage {
			t.Errorf("test %d: expected '%s' but got '%s'", i, tc.expectMessage, actual)
		}
	}
}

func TestPrometheus_Stop(t *testing.T) {
	err := Prometheus{}.Stop(nil)
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}
}

func TestPrometheus_Routes(t *testing.T) {
	for i, tc := range []struct {
		expectRoutes   int
		disableCollect bool
	}{
		{
			disableCollect: true,
		},
		{
			expectRoutes: 1,
		},
	} {
		mod := Prometheus{
			disableCollect: tc.disableCollect,
			registry:       prometheus.NewRegistry(),
		}

		routes, err := mod.Routes()
		if err != nil {
			t.Fatalf("test %d: expected no error but got: %v", i, err)
		}

		if tc.expectRoutes != len(routes) {
			t.Errorf("test %d: expected %d routes but got %d", i, tc.expectRoutes, len(routes))
		}
	}
}

// Interface guards.
var (
	_ gotenberg.Module          = (*ProtoModule)(nil)
	_ gotenberg.Validator       = (*ProtoValidator)(nil)
	_ gotenberg.Module          = (*ProtoValidator)(nil)
	_ gotenberg.MetricsProvider = (*ProtoMetricsProvider)(nil)
	_ gotenberg.Module          = (*ProtoMetricsProvider)(nil)
	_ gotenberg.Validator       = (*ProtoMetricsProvider)(nil)
)
