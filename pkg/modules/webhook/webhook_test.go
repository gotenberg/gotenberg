package webhook

import (
	"reflect"
	"testing"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func TestWebhook_Descriptor(t *testing.T) {
	descriptor := new(Webhook).Descriptor()

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
	for _, tc := range []struct {
		scenario          string
		disable           bool
		expectMiddlewares int
	}{
		{
			scenario:          "webhook disabled",
			disable:           true,
			expectMiddlewares: 0,
		},
		{
			scenario:          "webhook enabled",
			disable:           false,
			expectMiddlewares: 1,
		},
	} {
		mod := new(Webhook)
		mod.disable = tc.disable

		middlewares, err := mod.Middlewares()
		if err != nil {
			t.Fatalf("expected no error but got: %v", err)
		}

		if tc.expectMiddlewares != len(middlewares) {
			t.Errorf("expected %d middlewares but got %d", tc.expectMiddlewares, len(middlewares))
		}
	}
}
