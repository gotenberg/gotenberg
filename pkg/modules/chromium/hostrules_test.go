package chromium

import (
	"errors"
	"testing"
)

// The cases below come from Chromium's base/strings/pattern_unittest.cc and
// net/base/host_mapping_rules_unittest.cc, so that the pinning proxy maps
// hosts exactly as Chromium would.

func TestMatchHostPattern(t *testing.T) {
	for _, tc := range []struct {
		value   string
		pattern string
		expect  bool
	}{
		{"www.google.com", "*.com", true},
		{"www.google.com", "*", true},
		{"www.google.com", "www*.g*.org", false},
		{"Hello", "H?l?o", true},
		{"www.google.com", "http://*)", false},
		{"www.msn.com", "*.COM", false},
		{"Hello*1234", "He??o\\*1*", true},
		{"", "*.*", false},
		{"", "*", true},
		{"", "?", true},
		{"", "", true},
		{"Hello", "", false},
		{"Hello*", "Hello*", true},
		{"abcd", "*???", true},
		{"abcd", "???", false},
		{"abcb", "a*b", true},
		{"abcb", "a?b", false},
		{"heart: ♠", "*♠", true},
		{"heart: ♠.", "heart: ?.", true},
		{"invalid: \xef\xbf\xbe", "invalid: ?", true},
		{"Hello", "He********************************o", true},
		{"123456789012345678", "?????????????????*", true},
		{"aaaaaaaaaaab", "a*a*a*a*a*a*a*a*a*a*a*b", true},
	} {
		t.Run(tc.value+" "+tc.pattern, func(t *testing.T) {
			if got := matchHostPattern(tc.value, tc.pattern); got != tc.expect {
				t.Errorf("matchHostPattern(%q, %q) = %t, want %t", tc.value, tc.pattern, got, tc.expect)
			}
		})
	}
}

func TestHostResolverRules(t *testing.T) {
	for _, tc := range []struct {
		scenario   string
		rules      string
		host, port string
		expectHost string
		expectPort string
		expectErr  error
	}{
		{"no rules", "", "test", "1234", "test", "1234", nil},
		{"no matching rule", "map *.com baz , map *.net bar:60, EXCLUDE *.foo.com", "test", "1234", "test", "1234", nil},
		{"host and port", "map *.com baz , map *.net bar:60, EXCLUDE *.foo.com", "chrome.net", "80", "bar", "60", nil},
		{"host only", "map *.com baz , map *.net bar:60, EXCLUDE *.foo.com", "crack.com", "80", "baz", "80", nil},
		{"excluded", "map *.com baz , map *.net bar:60, EXCLUDE *.foo.com", "wtf.foo.com", "666", "wtf.foo.com", "666", nil},
		{"port-specific, no match", "map *.com:80 baz:111 , map *.com:443 blat:333, EXCLUDE *.foo.com", "test.com", "1234", "test.com", "1234", nil},
		{"port-specific 80", "map *.com:80 baz:111 , map *.com:443 blat:333, EXCLUDE *.foo.com", "crack.com", "80", "baz", "111", nil},
		{"port-specific 443", "map *.com:80 baz:111 , map *.com:443 blat:333, EXCLUDE *.foo.com", "wtf.com", "443", "blat", "333", nil},
		{"port-specific, excluded", "map *.com:80 baz:111 , map *.com:443 blat:333, EXCLUDE *.foo.com", "wtf.foo.com", "443", "wtf.foo.com", "443", nil},
		{"IPv6 replacement", "MAP initial.test [2345:6789::0abc]:1112", "initial.test", "111", "2345:6789::0abc", "1112", nil},
		{"IPv6 host matched with its port", "MAP [::1]:80 example.test", "::1", "80", "example.test", "80", nil},
		{"first matching rule wins", "MAP a.test 10.0.0.1, MAP a.test 10.0.0.2", "a.test", "443", "10.0.0.1", "443", nil},
		{"pattern case", "MAP EXAMPLE.TEST 127.0.0.1", "example.test", "80", "127.0.0.1", "80", nil},
		{"trailing comma", "MAP a.test 127.0.0.1,", "a.test", "80", "127.0.0.1", "80", nil},
		{"^NOTFOUND", "MAP * ^NOTFOUND , EXCLUDE localhost", "example.test", "443", "", "", errHostNotFound},
		{"~NOTFOUND", "MAP * ~NOTFOUND", "example.test", "443", "", "", errHostNotFound},
		{"NOTFOUND excluded", "MAP * ~NOTFOUND , EXCLUDE localhost", "localhost", "80", "localhost", "80", nil},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			rules, err := parseHostResolverRules(tc.rules)
			if err != nil {
				t.Fatalf("parse %q: %v", tc.rules, err)
			}

			host, port, err := rules.rewrite(tc.host, tc.port)
			if !errors.Is(err, tc.expectErr) {
				t.Fatalf("expected error %v, got %v", tc.expectErr, err)
			}
			if host != tc.expectHost || port != tc.expectPort {
				t.Errorf("rewrite(%q, %q) = %q, %q, want %q, %q", tc.host, tc.port, host, port, tc.expectHost, tc.expectPort)
			}
		})
	}
}

func TestParseHostResolverRulesInvalid(t *testing.T) {
	for _, rules := range []string{
		"xyz",
		"EXCLUDE",
		"EXCLUDE foo bar",
		"INCLUDE",
		"INCLUDE x",
		"INCLUDE x :10",
		"MAP a.test",
		"MAP  a.test b.test",
		"MAP a.test b.test:",
		"MAP a.test b.test:65536",
		"MAP a.test b.test:http",
		"MAP a.test :80",
		"MAP a.test user@b.test",
		"MAP a.test [::1",
		"MAP a.test [example]:80",
		"MAP a.test 127.0.0.1, xyz",
	} {
		t.Run(rules, func(t *testing.T) {
			_, err := parseHostResolverRules(rules)
			if err == nil {
				t.Errorf("expected an error for %q", rules)
			}
		})
	}
}
