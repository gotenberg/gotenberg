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
	r.WithCustomHeader(customHeaderCanonicalKey, customHeaderValue)
	r.WithCustomHeader("Bar", "Bar")
	expected := map[string]string{
		customHeaderCanonicalRealKey: customHeaderValue,
	}
	notExpected := map[string]string{
		customHeaderCanonicalKey: customHeaderValue,
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
	r.WithCustomHeader(customHeaderCanonicalKey, customHeaderValue)
	r.WithCustomHeader("Bar", "Bar")
	expected := map[string]string{
		customHeaderCanonicalRealKey: customHeaderValue,
	}
	notExpected := map[string]string{
		customHeaderCanonicalKey: customHeaderValue,
	}
	v := WebhookURLCustomHeaders(r)
	assert.Equal(t, expected, v)
	assert.NotEqual(t, notExpected, v)
}
