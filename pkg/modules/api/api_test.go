package api

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestApi_Validate_Auth(t *testing.T) {
	base := func() *Api {
		return &Api{port: 3000, rootPath: "/", correlationIdHeader: "Gotenberg-Trace"}
	}

	for _, tc := range []struct {
		scenario string
		mutate   func(*Api)
		wantErr  string // substring expected in the error, "" means no error
	}{
		{"no auth", func(*Api) {}, ""},
		{"basic auth only", func(a *Api) { a.basicAuthUsername = "foo" }, ""},
		{
			"oidc auth valid",
			func(a *Api) {
				a.oidcEnabled = true
				a.oidcIssuer = "https://tenant.example.com/"
				a.oidcAudience = "gotenberg"
			},
			"",
		},
		{
			"basic and oidc are mutually exclusive",
			func(a *Api) {
				a.basicAuthUsername = "foo"
				a.oidcEnabled = true
				a.oidcIssuer = "https://tenant.example.com/"
				a.oidcAudience = "gotenberg"
			},
			"cannot both be enabled",
		},
		{
			"oidc missing issuer",
			func(a *Api) { a.oidcEnabled = true; a.oidcAudience = "gotenberg" },
			"issuer must not be empty",
		},
		{
			"oidc missing audience",
			func(a *Api) { a.oidcEnabled = true; a.oidcIssuer = "https://tenant.example.com/" },
			"audience must not be empty",
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			a := base()
			tc.mutate(a)

			err := a.Validate()

			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want a substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestApi_warnInsecureDebugRoute(t *testing.T) {
	for _, tc := range []struct {
		scenario     string
		api          Api
		expectWarn   bool
		expectFields []string
	}{
		{
			scenario:     "debug route on with no auth warns",
			api:          Api{enableDebugRoute: true},
			expectWarn:   true,
			expectFields: []string{"--api-enable-debug-route", "API_ENABLE_DEBUG_ROUTE", "--api-enable-basic-auth", "API_ENABLE_BASIC_AUTH", "--api-enable-oidc-auth", "API_ENABLE_OIDC_AUTH"},
		},
		{
			scenario:   "debug route off is silent",
			api:        Api{enableDebugRoute: false},
			expectWarn: false,
		},
		{
			scenario:   "basic auth silences it",
			api:        Api{enableDebugRoute: true, basicAuthUsername: "foo"},
			expectWarn: false,
		},
		{
			scenario:   "oidc silences it",
			api:        Api{enableDebugRoute: true, oidcEnabled: true},
			expectWarn: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			buf := new(bytes.Buffer)
			tc.api.logger = slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

			tc.api.warnInsecureDebugRoute()

			logged := buf.String()
			if !tc.expectWarn {
				if logged != "" {
					t.Fatalf("expected no warning, got: %s", logged)
				}
				return
			}

			if logged == "" {
				t.Fatal("expected a warning, got none")
			}
			// Every flag named must carry its environment variable.
			for _, want := range tc.expectFields {
				if !strings.Contains(logged, want) {
					t.Fatalf("warning does not mention %q: %s", want, logged)
				}
			}
			if strings.Contains(logged, "—") {
				t.Fatalf("warning must not contain an em dash: %s", logged)
			}
		})
	}
}

// The warning reads a.logger, which is nil until Provision assigns it.
func TestApi_warnInsecureDebugRoute_NilLoggerDoesNotPanic(t *testing.T) {
	api := Api{enableDebugRoute: true}
	api.warnInsecureDebugRoute()
}
