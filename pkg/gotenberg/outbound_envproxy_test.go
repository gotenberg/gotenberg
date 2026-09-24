package gotenberg

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
)

func TestValidateEnvironmentProxyVariables(t *testing.T) {
	for _, tc := range []struct {
		name    string
		env     map[string]string
		wantErr bool
		// wantIn is a substring the error must name, so that an operator can
		// find the offending variable.
		wantIn string
	}{
		{
			name: "nothing set",
			env:  map[string]string{},
		},
		{
			name: "well formed URL",
			env:  map[string]string{"HTTP_PROXY": "http://proxy.example.com:3128"},
		},
		{
			name: "credentials are accepted",
			env:  map[string]string{"HTTPS_PROXY": "http://user:password@proxy.example.com:3128"},
		},
		{
			name: "bare host and port is accepted, as httpproxy prefixes a scheme",
			env:  map[string]string{"HTTP_PROXY": "proxy.example.com:3128"},
		},
		{
			name: "socks5 is accepted",
			env:  map[string]string{"ALL_PROXY": "socks5://proxy.example.com:1080"},
		},
		{
			name: "lowercase variables are checked too",
			env:  map[string]string{"http_proxy": "http://proxy.example.com:3128"},
		},
		{
			name:    "unparseable URL",
			env:     map[string]string{"HTTP_PROXY": "http://proxy.example.com:3128/%zz"},
			wantErr: true,
			wantIn:  "HTTP_PROXY",
		},
		{
			name:    "the failing variable is named",
			env:     map[string]string{"HTTPS_PROXY": "://%zz"},
			wantErr: true,
			wantIn:  "HTTPS_PROXY",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, name := range environmentProxyVariables {
				t.Setenv(name, "")
			}
			for name, value := range tc.env {
				t.Setenv(name, value)
			}

			err := ValidateEnvironmentProxyVariables()
			if tc.wantErr && err == nil {
				t.Fatal("expected an error, got none")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantIn != "" && !strings.Contains(err.Error(), tc.wantIn) {
				t.Errorf("error %q does not name %q", err, tc.wantIn)
			}
		})
	}
}

// TestValidateEnvironmentProxyVariables_DoesNotLeakCredentials pins that a
// proxy URL, which may embed a password, never reaches the error text.
func TestValidateEnvironmentProxyVariables_DoesNotLeakCredentials(t *testing.T) {
	for _, name := range environmentProxyVariables {
		t.Setenv(name, "")
	}
	t.Setenv("HTTP_PROXY", "http://admin:hunter2@proxy.example.com:3128/%zz")

	err := ValidateEnvironmentProxyVariables()
	if err == nil {
		t.Fatal("expected an error, got none")
	}
	if strings.Contains(err.Error(), "hunter2") {
		t.Errorf("error leaks the proxy password: %q", err)
	}
	if strings.Contains(err.Error(), "admin") {
		t.Errorf("error leaks the proxy username: %q", err)
	}
}

// TestNewOutboundHttpClient_EnvironmentProxyPinsDirectHops is the regression
// test for the dial-pinning gap: with the environment proxy enabled but no
// proxy applicable to the request, the dial must still go through the pinning
// dialer rather than a plain one.
//
// The request targets a hostname that only the stub resolver knows, so a plain
// dial would hand that unresolvable name to the OS and fail. Only a pinned dial,
// which substitutes the address resolved during validation, can connect.
// See https://github.com/gotenberg/gotenberg/issues/1592.
func TestNewOutboundHttpClient_EnvironmentProxyPinsDirectHops(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	_, port, err := net.SplitHostPort(strings.TrimPrefix(srv.URL, "http://"))
	if err != nil {
		t.Fatalf("split server address: %v", err)
	}

	const host = "pinned-only.invalid"

	// NO_PROXY covers the destination, so httpproxy declines it and the
	// transport dials directly. That direct dial is the hop that used to lose
	// pinning.
	for _, name := range environmentProxyVariables {
		t.Setenv(name, "")
	}
	t.Setenv("HTTP_PROXY", "http://proxy.invalid:3128")
	t.Setenv("NO_PROXY", host)

	withStubResolver(t, func(string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
	})

	client := NewOutboundHttpClient(0, nil, nil, true)
	rt, ok := client.Transport.(*outboundRoundTripper)
	if !ok {
		t.Fatalf("transport is %T, want *outboundRoundTripper", client.Transport)
	}
	if rt.proxyFunc == nil {
		t.Fatal("proxyFunc is nil, want the environment proxy to be resolved per request")
	}

	resp, err := client.Get("http://" + net.JoinHostPort(host, port))
	if err != nil {
		t.Fatalf("GET failed, so the direct hop was not pinned: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
}
