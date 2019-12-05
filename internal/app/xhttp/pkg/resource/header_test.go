package resource

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestRemoteURLCustomHeaders(t *testing.T) {
	const resourceDirectoryName string = "foo"
	logger := test.DebugLogger()
	r, err := New(logger, resourceDirectoryName)
	assert.Nil(t, err)
	// should find the custom header.
	customHeaderValue := "bar"
	customHeaderCanonicalRealKey := "Foo"
	customHeaderCanonicalKey := http.CanonicalHeaderKey(fmt.Sprintf("%s%s", RemoteURLCustomHeaderCanonicalBaseKey, customHeaderCanonicalRealKey))
	r.WithCustomHeader(customHeaderCanonicalKey, []string{customHeaderValue})
	r.WithCustomHeader("Bar", []string{"Bar"})
	expected := map[string][]string{
		customHeaderCanonicalRealKey: []string{
			customHeaderValue,
		},
	}
	notExpected := map[string][]string{
		customHeaderCanonicalKey: []string{
			customHeaderValue,
		},
	}
	v := RemoteURLCustomHeaders(r)
	assert.Equal(t, expected, v)
	assert.NotEqual(t, notExpected, v)
}

func TestWebhookURLCustomHeaders(t *testing.T) {
	const resourceDirectoryName string = "foo"
	logger := test.DebugLogger()
	r, err := New(logger, resourceDirectoryName)
	assert.Nil(t, err)
	// should find the custom header.
	customHeaderValue := "bar"
	customHeaderCanonicalRealKey := "Foo"
	customHeaderCanonicalKey := http.CanonicalHeaderKey(fmt.Sprintf("%s%s", WebhookURLCustomHeaderCanonicalBaseKey, customHeaderCanonicalRealKey))
	r.WithCustomHeader(customHeaderCanonicalKey, []string{customHeaderValue})
	r.WithCustomHeader("Bar", []string{"Bar"})
	expected := map[string][]string{
		customHeaderCanonicalRealKey: []string{
			customHeaderValue,
		},
	}
	notExpected := map[string][]string{
		customHeaderCanonicalKey: []string{
			customHeaderValue,
		},
	}
	v := WebhookURLCustomHeaders(r)
	assert.Equal(t, expected, v)
	assert.NotEqual(t, notExpected, v)
}
