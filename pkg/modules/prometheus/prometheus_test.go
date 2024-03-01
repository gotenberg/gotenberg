package prometheus

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func TestPrometheus_Descriptor(t *testing.T) {
	descriptor := new(Prometheus).Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(Prometheus))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestPrometheus_Provision(t *testing.T) {
	for _, tc := range []struct {
		scenario      string
		ctx           *gotenberg.Context
		expectMetrics []gotenberg.Metric
		expectError   bool
	}{
		{
			scenario: "disable collect",
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
			expectError: false,
		},
		{
			scenario: "invalid metrics provider",
			ctx: func() *gotenberg.Context {
				mod := &struct {
					gotenberg.ModuleMock
					gotenberg.ValidatorMock
					gotenberg.MetricsProviderMock
				}{}
				mod.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.ValidateMock = func() error {
					return errors.New("foo")
				}
				mod.MetricsMock = func() ([]gotenberg.Metric, error) {
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
			expectError: true,
		},
		{
			scenario: "invalid metrics from metrics provider",
			ctx: func() *gotenberg.Context {
				mod := &struct {
					gotenberg.ModuleMock
					gotenberg.ValidatorMock
					gotenberg.MetricsProviderMock
				}{}
				mod.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.ValidateMock = func() error {
					return nil
				}
				mod.MetricsMock = func() ([]gotenberg.Metric, error) {
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
			expectError: true,
		},
		{
			scenario: "provision success",
			ctx: func() *gotenberg.Context {
				mod := &struct {
					gotenberg.ModuleMock
					gotenberg.ValidatorMock
					gotenberg.MetricsProviderMock
				}{}
				mod.DescriptorMock = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.ValidateMock = func() error {
					return nil
				}
				mod.MetricsMock = func() ([]gotenberg.Metric, error) {
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
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			mod := new(Prometheus)
			err := mod.Provision(tc.ctx)

			if !reflect.DeepEqual(mod.metrics, tc.expectMetrics) {
				t.Fatalf("expected metrics %+v, but got: %+v", tc.expectMetrics, mod.metrics)
			}

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestPrometheus_Validate(t *testing.T) {
	for _, tc := range []struct {
		scenario       string
		namespace      string
		metrics        []gotenberg.Metric
		disableCollect bool
		expectError    bool
	}{
		{
			scenario:       "collect disabled",
			namespace:      "foo",
			disableCollect: true,
			expectError:    false,
		},
		{
			scenario:       "empty namespace",
			namespace:      "",
			disableCollect: false,
			expectError:    true,
		},
		{
			scenario:  "empty metric name",
			namespace: "foo",
			metrics: []gotenberg.Metric{
				{
					Name: "",
				},
			},
			disableCollect: false,
			expectError:    true,
		},
		{
			scenario:  "nil read metric method",
			namespace: "foo",
			metrics: []gotenberg.Metric{
				{
					Name: "foo",
					Read: nil,
				},
			},
			disableCollect: false,
			expectError:    true,
		},
		{
			scenario:  "already registered metric",
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
			disableCollect: false,
			expectError:    true,
		},
		{
			scenario:  "validate success",
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
			disableCollect: false,
			expectError:    false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			mod := &Prometheus{
				namespace:      tc.namespace,
				metrics:        tc.metrics,
				disableCollect: tc.disableCollect,
			}
			err := mod.Validate()

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestPrometheus_Start(t *testing.T) {
	for _, tc := range []struct {
		scenario       string
		metrics        []gotenberg.Metric
		disableCollect bool
	}{
		{
			scenario:       "collect disabled",
			disableCollect: true,
		},
		{
			scenario: "start success",
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
		t.Run(tc.scenario, func(t *testing.T) {
			mod := &Prometheus{
				namespace:      "foo",
				interval:       time.Duration(1) * time.Second,
				metrics:        tc.metrics,
				disableCollect: tc.disableCollect,
				registry:       prometheus.NewRegistry(),
			}

			err := mod.Start()
			if err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestPrometheus_StartupMessage(t *testing.T) {
	mod := new(Prometheus)

	mod.disableCollect = true
	disableCollectMsg := mod.StartupMessage()

	mod.disableCollect = false
	noDisableCollectMsg := mod.StartupMessage()

	if disableCollectMsg == noDisableCollectMsg {
		t.Errorf("expected differrent startup messages if collect is disabled or not, but got '%s'", disableCollectMsg)
	}
}

func TestPrometheus_Stop(t *testing.T) {
	err := new(Prometheus).Stop(nil)
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}
}

func TestPrometheus_Routes(t *testing.T) {
	for _, tc := range []struct {
		scenario       string
		disableCollect bool
		expectRoutes   int
	}{
		{
			scenario:       "collect disabled",
			disableCollect: true,
			expectRoutes:   0,
		},
		{
			scenario:       "routes not disabled",
			disableCollect: false,
			expectRoutes:   1,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			mod := &Prometheus{
				disableCollect: tc.disableCollect,
				registry:       prometheus.NewRegistry(),
			}

			routes, err := mod.Routes()
			if err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectRoutes != len(routes) {
				t.Errorf("expected %d routes but got %d", tc.expectRoutes, len(routes))
			}
		})
	}
}
