package api

import (
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
