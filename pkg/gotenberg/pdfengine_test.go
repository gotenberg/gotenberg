package gotenberg

import "testing"

func TestOptimizeOptions_HasImagesOptimization(t *testing.T) {
	for _, tc := range []struct {
		scenario string
		options  OptimizeOptions
		expect   bool
	}{
		{
			scenario: "no images optimization",
			options:  OptimizeOptions{},
			expect:   false,
		},
		{
			scenario: "images optimization (image quality only)",
			options:  OptimizeOptions{ImageQuality: 90},
			expect:   true,
		},
		{
			scenario: "images optimization (image max resolution only)",
			options:  OptimizeOptions{MaxImageResolution: 75},
			expect:   true,
		},
		{
			scenario: "images optimization (all)",
			options:  OptimizeOptions{ImageQuality: 90, MaxImageResolution: 75},
			expect:   true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			got := tc.options.HasImagesOptimization()
			if got != tc.expect {
				t.Errorf("expected %t but got %t", tc.expect, got)
			}
		})
	}
}
