package webhook

import (
	"reflect"
	"testing"
	"time"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
)

func TestWebhook_Descriptor(t *testing.T) {
	descriptor := Webhook{}.Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(Webhook))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestWebhook_Provision(t *testing.T) {
	mod := new(Webhook)
	ctx := gotenberg.NewContext(
		gotenberg.ParsedFlags{
			FlagSet: new(Webhook).Descriptor().FlagSet,
		},
		nil,
	)

	err := mod.Provision(ctx)
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}
}

func TestWebhook_Middlewares(t *testing.T) {
	for i, tc := range []struct {
		expectMiddlewares int
		disable           bool
	}{
		{
			expectMiddlewares: 1,
		},
		{
			disable: true,
		},
	} {
		mod := new(Webhook)
		mod.disable = tc.disable

		middlewares, err := mod.Middlewares()
		if err != nil {
			t.Fatalf("test %d: expected no error but got: %v", i, err)
		}

		if tc.expectMiddlewares != len(middlewares) {
			t.Errorf("test %d: expected %d middlewares but got %d", i, tc.expectMiddlewares, len(middlewares))
		}
	}
}

func TestWebhook_AddGraceDuration(t *testing.T) {
	for i, tc := range []struct {
		maxRetry       int
		retryMaxWait   time.Duration
		expectDuration time.Duration
		disable        bool
	}{
		{
			maxRetry:       3,
			retryMaxWait:   time.Duration(1) * time.Second,
			expectDuration: time.Duration(3) * time.Second,
		},
		{
			disable: true,
		},
	} {
		mod := new(Webhook)
		mod.maxRetry = tc.maxRetry
		mod.retryMaxWait = tc.retryMaxWait
		mod.disable = tc.disable

		actual := mod.AddGraceDuration()
		if actual != tc.expectDuration {
			t.Errorf("test %d: expected '%s' but got '%s'", i, tc.expectDuration, actual)
		}
	}
}
