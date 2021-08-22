package pdfengines

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"go.uber.org/zap"
)

func TestNewMultiPDFEngines(t *testing.T) {
	engine1 := &ProtoPDFEngine{
		merge: func(_ context.Context, _ *zap.Logger, _ []string, _ string) error {
			return nil
		},
		convert: func(_ context.Context, _ *zap.Logger, _, _, _ string) error {
			return nil
		},
	}

	engine2 := &ProtoPDFEngine{
		merge: func(_ context.Context, _ *zap.Logger, _ []string, _ string) error {
			return errors.New("foo")
		},
		convert: func(_ context.Context, _ *zap.Logger, _, _, _ string) error {
			return errors.New("foo")
		},
	}

	multi := newMultiPDFEngines(engine1, engine2)

	if len(multi.engines) != 2 {
		t.Fatalf("expected %d engines but got %d", 2, len(multi.engines))
	}

	if !reflect.DeepEqual(engine1, multi.engines[0]) {
		t.Errorf("expected %v, but got: %v", engine1, multi.engines[0])
	}

	if !reflect.DeepEqual(engine2, multi.engines[1]) {
		t.Errorf("expected %v, but got: %v", engine2, multi.engines[1])
	}
}

func TestMultiPDFEngines_Merge(t *testing.T) {
	for i, tc := range []struct {
		ctx       context.Context
		engines   []gotenberg.PDFEngine
		expectErr bool
	}{
		{
			ctx: context.TODO(),
			engines: func() []gotenberg.PDFEngine {
				return []gotenberg.PDFEngine{
					ProtoPDFEngine{
						merge: func(_ context.Context, _ *zap.Logger, _ []string, _ string) error {
							return nil
						},
					},
				}
			}(),
		},
		{
			ctx: context.TODO(),
			engines: func() []gotenberg.PDFEngine {
				return []gotenberg.PDFEngine{
					ProtoPDFEngine{
						merge: func(_ context.Context, _ *zap.Logger, _ []string, _ string) error {
							return errors.New("foo")
						},
					},
					ProtoPDFEngine{
						merge: func(_ context.Context, _ *zap.Logger, _ []string, _ string) error {
							return nil
						},
					},
				}
			}(),
		},
		{
			ctx: context.TODO(),
			engines: func() []gotenberg.PDFEngine {
				return []gotenberg.PDFEngine{
					ProtoPDFEngine{
						merge: func(_ context.Context, _ *zap.Logger, _ []string, _ string) error {
							return errors.New("foo")
						},
					},
					ProtoPDFEngine{
						merge: func(_ context.Context, _ *zap.Logger, _ []string, _ string) error {
							return errors.New("bar")
						},
					},
				}
			}(),
			expectErr: true,
		},
		{
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.TODO())
				defer cancel()

				return ctx
			}(),
			engines: func() []gotenberg.PDFEngine {
				return []gotenberg.PDFEngine{
					ProtoPDFEngine{
						merge: func(_ context.Context, _ *zap.Logger, _ []string, _ string) error {
							return nil
						},
					},
				}
			}(),
			expectErr: true,
		},
	} {
		multi := newMultiPDFEngines(tc.engines...)
		err := multi.Merge(tc.ctx, nil, nil, "")

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestMultiPDFEngines_Convert(t *testing.T) {
	for i, tc := range []struct {
		ctx       context.Context
		engines   []gotenberg.PDFEngine
		expectErr bool
	}{
		{
			ctx: context.TODO(),
			engines: func() []gotenberg.PDFEngine {
				return []gotenberg.PDFEngine{
					ProtoPDFEngine{
						convert: func(_ context.Context, _ *zap.Logger, _, _, _ string) error {
							return nil
						},
					},
				}
			}(),
		},
		{
			ctx: context.TODO(),
			engines: func() []gotenberg.PDFEngine {
				return []gotenberg.PDFEngine{
					ProtoPDFEngine{
						convert: func(_ context.Context, _ *zap.Logger, _, _, _ string) error {
							return errors.New("foo")
						},
					},
					ProtoPDFEngine{
						convert: func(_ context.Context, _ *zap.Logger, _, _, _ string) error {
							return nil
						},
					},
				}
			}(),
		},
		{
			ctx: context.TODO(),
			engines: func() []gotenberg.PDFEngine {
				return []gotenberg.PDFEngine{
					ProtoPDFEngine{
						convert: func(_ context.Context, _ *zap.Logger, _, _, _ string) error {
							return errors.New("foo")
						},
					},
					ProtoPDFEngine{
						convert: func(_ context.Context, _ *zap.Logger, _, _, _ string) error {
							return errors.New("bar")
						},
					},
				}
			}(),
			expectErr: true,
		},
		{
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.TODO())
				defer cancel()

				return ctx
			}(),
			engines: func() []gotenberg.PDFEngine {
				return []gotenberg.PDFEngine{
					ProtoPDFEngine{
						convert: func(_ context.Context, _ *zap.Logger, _, _, _ string) error {
							return nil
						},
					},
				}
			}(),
			expectErr: true,
		},
	} {
		multi := newMultiPDFEngines(tc.engines...)
		err := multi.Convert(tc.ctx, nil, "", "", "")

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}
