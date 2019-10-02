package test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// AssertStatusCode checks if the given request
// returns the expected status code.
func AssertStatusCode(t *testing.T, expectedStatusCode int, srv http.Handler, req *http.Request) {
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	assert.Equal(t, expectedStatusCode, rec.Code)
}
