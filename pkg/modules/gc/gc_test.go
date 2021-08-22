package gc

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"go.uber.org/zap"
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

type ProtoGarbageCollectorGraceDurationModifier struct {
	ProtoValidator
	graceDuration func() time.Duration
}

func (mod ProtoGarbageCollectorGraceDurationModifier) GraceDuration() time.Duration {
	return mod.graceDuration()
}

type ProtoGarbageCollectorExcludeSubstrModifier struct {
	ProtoValidator
	excludeSubstr func() []string
}

func (mod ProtoGarbageCollectorExcludeSubstrModifier) ExcludeSubstr() []string {
	return mod.excludeSubstr()
}

type ProtoLoggerProvider struct {
	ProtoModule
	logger func(mod gotenberg.Module) (*zap.Logger, error)
}

func (factory ProtoLoggerProvider) Logger(mod gotenberg.Module) (*zap.Logger, error) {
	return factory.logger(mod)
}

func TestGarbageCollector_Descriptor(t *testing.T) {
	descriptor := GarbageCollector{}.Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(GarbageCollector))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestGarbageCollector_Provision(t *testing.T) {
	for i, tc := range []struct {
		ctx                 *gotenberg.Context
		expectGraceDuration time.Duration
		expectExcludeSubstr []string
		expectErr           bool
	}{
		{
			ctx: func() *gotenberg.Context {
				mod := struct {
					ProtoGarbageCollectorGraceDurationModifier
				}{}
				mod.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.validate = func() error { return errors.New("foo") }

				return gotenberg.NewContext(gotenberg.ParsedFlags{}, []gotenberg.ModuleDescriptor{
					mod.Descriptor(),
				})
			}(),
			expectErr: true,
		},
		{
			ctx: func() *gotenberg.Context {
				mod := struct {
					ProtoGarbageCollectorExcludeSubstrModifier
				}{}
				mod.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.validate = func() error { return errors.New("foo") }

				return gotenberg.NewContext(gotenberg.ParsedFlags{}, []gotenberg.ModuleDescriptor{
					mod.Descriptor(),
				})
			}(),
			expectErr: true,
		},
		{
			ctx:       gotenberg.NewContext(gotenberg.ParsedFlags{}, make([]gotenberg.ModuleDescriptor, 0)),
			expectErr: true,
		},
		{
			ctx: func() *gotenberg.Context {
				mod := struct {
					ProtoLoggerProvider
				}{}
				mod.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.logger = func(mod gotenberg.Module) (*zap.Logger, error) { return nil, errors.New("foo") }

				return gotenberg.NewContext(gotenberg.ParsedFlags{}, []gotenberg.ModuleDescriptor{
					mod.Descriptor(),
				})
			}(),
			expectErr: true,
		},
		{
			ctx: func() *gotenberg.Context {
				mod := struct {
					ProtoLoggerProvider
				}{}
				mod.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod }}
				}
				mod.logger = func(mod gotenberg.Module) (*zap.Logger, error) { return zap.NewNop(), nil }

				return gotenberg.NewContext(gotenberg.ParsedFlags{}, []gotenberg.ModuleDescriptor{
					mod.Descriptor(),
				})
			}(),
		},
		{
			ctx: func() *gotenberg.Context {
				mod1 := struct {
					ProtoGarbageCollectorGraceDurationModifier
				}{}
				mod1.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod1 }}
				}
				mod1.graceDuration = func() time.Duration { return time.Duration(10) * time.Second }
				mod1.validate = func() error { return nil }

				mod2 := struct {
					ProtoGarbageCollectorGraceDurationModifier
				}{}
				mod2.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module { return mod2 }}
				}
				mod2.graceDuration = func() time.Duration { return time.Duration(20) * time.Second }
				mod2.validate = func() error { return nil }

				mod3 := struct {
					ProtoLoggerProvider
				}{}
				mod3.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "baz", New: func() gotenberg.Module { return mod3 }}
				}
				mod3.logger = func(mod gotenberg.Module) (*zap.Logger, error) { return zap.NewNop(), nil }

				return gotenberg.NewContext(gotenberg.ParsedFlags{}, []gotenberg.ModuleDescriptor{
					mod1.Descriptor(),
					mod2.Descriptor(),
					mod3.Descriptor(),
				})
			}(),
			expectGraceDuration: time.Duration(20) * time.Second,
		},
		{
			ctx: func() *gotenberg.Context {
				mod1 := struct {
					ProtoGarbageCollectorExcludeSubstrModifier
				}{}
				mod1.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "foo", New: func() gotenberg.Module { return mod1 }}
				}
				mod1.excludeSubstr = func() []string { return []string{"foo"} }
				mod1.validate = func() error { return nil }

				mod2 := struct {
					ProtoGarbageCollectorExcludeSubstrModifier
				}{}
				mod2.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "bar", New: func() gotenberg.Module { return mod2 }}
				}
				mod2.excludeSubstr = func() []string { return []string{"bar"} }
				mod2.validate = func() error { return nil }

				mod3 := struct {
					ProtoLoggerProvider
				}{}
				mod3.descriptor = func() gotenberg.ModuleDescriptor {
					return gotenberg.ModuleDescriptor{ID: "baz", New: func() gotenberg.Module { return mod3 }}
				}
				mod3.logger = func(mod gotenberg.Module) (*zap.Logger, error) { return zap.NewNop(), nil }

				return gotenberg.NewContext(gotenberg.ParsedFlags{}, []gotenberg.ModuleDescriptor{
					mod1.Descriptor(),
					mod2.Descriptor(),
					mod3.Descriptor(),
				})
			}(),
			expectExcludeSubstr: func() []string {
				expect := strings.Split(os.Getenv("GC_EXCLUDE_SUBSTR"), ",")
				return append(expect, "foo", "bar")
			}(),
		},
	} {
		mod := new(GarbageCollector)
		err := mod.Provision(tc.ctx)

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}

		if tc.expectGraceDuration != 0 && tc.expectGraceDuration != mod.graceDuration {
			t.Errorf("test %d: expected grace duration of '%s' but got '%s'", i, tc.expectGraceDuration, mod.graceDuration)
		}

		if tc.expectExcludeSubstr != nil && !reflect.DeepEqual(tc.expectExcludeSubstr, mod.excludeSubstr) {
			t.Errorf("test %d: expected exclude substr '%s' but got '%s'", i, tc.expectExcludeSubstr, mod.excludeSubstr)
		}
	}
}

func TestGarbageCollector_Start(t *testing.T) {
	mod := new(GarbageCollector)
	mod.logger = zap.NewNop()

	path, err := gotenberg.MkdirAll()
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	mod.rootPath = path

	err = mod.Start()
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	time.Sleep(time.Duration(2) * time.Second)
	mod.ticker.Stop()
	mod.done <- true
}

func TestGarbageCollector_collect(t *testing.T) {
	for i, tc := range []struct {
		gc              *GarbageCollector
		expectNotExists []string
		expectExists    []string
		force           bool
	}{
		{
			gc: func() *GarbageCollector {
				mod := new(GarbageCollector)
				mod.logger = zap.NewNop()

				path, err := gotenberg.MkdirAll()
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				mod.rootPath = path

				err = os.WriteFile(path+"/foo", []byte{1}, 0755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				mod.excludeSubstr = []string{
					"foo",
				}

				return mod
			}(),
			expectExists: []string{
				"/foo",
			},
		},
		{
			gc: func() *GarbageCollector {
				mod := new(GarbageCollector)
				mod.logger = zap.NewNop()

				path, err := gotenberg.MkdirAll()
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				mod.rootPath = path

				err = os.WriteFile(path+"/foo", []byte{1}, 0755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				err = os.MkdirAll(path+"/bar", 0755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return mod
			}(),
			expectNotExists: []string{
				"/foo",
				"/bar",
			},
			force: true,
		},
		{
			gc: func() *GarbageCollector {
				mod := new(GarbageCollector)
				mod.logger = zap.NewNop()
				mod.graceDuration = time.Duration(10) * time.Second

				path, err := gotenberg.MkdirAll()
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				mod.rootPath = path

				err = os.WriteFile(path+"/foo", []byte{1}, 0755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				newTime := time.Now().Add(-time.Duration(20) * time.Second)
				err = os.Chtimes(path+"/foo", newTime, newTime)

				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				err = os.WriteFile(path+"/bar", []byte{1}, 0755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				newTime = time.Now().Add(time.Duration(10) * time.Second)
				err = os.Chtimes(path+"/bar", newTime, newTime)

				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return mod
			}(),
			expectNotExists: []string{
				"/foo",
			},
			expectExists: []string{
				"/bar",
			},
		},
	} {
		tc.gc.collect(tc.force)

		for _, name := range tc.expectNotExists {
			path := tc.gc.rootPath + name
			_, err := os.Stat(path)
			if !os.IsNotExist(err) {
				t.Errorf("test %d: expected '%s' not to exist but got: %v", i, path, err)
			}
		}

		for _, name := range tc.expectExists {
			path := tc.gc.rootPath + name
			_, err := os.Stat(path)
			if os.IsNotExist(err) {
				t.Errorf("test %d: expected '%s' to exist but got: %v", i, path, err)
			}
		}

		err := os.RemoveAll(tc.gc.rootPath)
		if err != nil {
			t.Fatalf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestGarbageCollector_StartupMessage(t *testing.T) {
	actual := new(GarbageCollector).StartupMessage()
	expect := ""

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestGarbageCollector_Stop(t *testing.T) {
	for i, tc := range []struct {
		timeout   time.Duration
		expectErr bool
	}{
		{
			expectErr: true,
		},
		{
			timeout: time.Duration(1) * time.Nanosecond,
		},
	} {
		func() {
			mod := new(GarbageCollector)
			mod.logger = zap.NewNop()

			path, err := gotenberg.MkdirAll()
			if err != nil {
				t.Fatalf("test %d: expected no error but got: %v", i, err)
			}

			mod.rootPath = path

			err = mod.Start()
			if err != nil {
				t.Fatalf("test %d: expected no error but got: %v", i, err)
			}

			if tc.timeout == 0 {
				err = mod.Stop(context.TODO())
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
				defer cancel()

				err = mod.Stop(ctx)
			}

			if tc.expectErr && err == nil {
				t.Errorf("test %d: expected error but got: %v", i, err)
			}

			if !tc.expectErr && err != nil {
				t.Errorf("test %d: expected no error but got: %v", i, err)
			}
		}()
	}
}

// Interface guards.
var (
	_ gotenberg.Module                      = (*ProtoModule)(nil)
	_ gotenberg.Validator                   = (*ProtoValidator)(nil)
	_ GarbageCollectorGraceDurationModifier = (*ProtoGarbageCollectorGraceDurationModifier)(nil)
	_ gotenberg.Module                      = (*ProtoGarbageCollectorGraceDurationModifier)(nil)
	_ gotenberg.Validator                   = (*ProtoGarbageCollectorGraceDurationModifier)(nil)
	_ GarbageCollectorExcludeSubstrModifier = (*ProtoGarbageCollectorExcludeSubstrModifier)(nil)
	_ gotenberg.Module                      = (*ProtoGarbageCollectorExcludeSubstrModifier)(nil)
	_ gotenberg.Validator                   = (*ProtoGarbageCollectorExcludeSubstrModifier)(nil)
	_ gotenberg.LoggerProvider              = (*ProtoLoggerProvider)(nil)
	_ gotenberg.Module                      = (*ProtoLoggerProvider)(nil)
)
