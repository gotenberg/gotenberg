package pdfcpu

import (
	"reflect"
	"sort"
	"testing"
)

func TestDigitSuffixSort(t *testing.T) {
	for _, tc := range []struct {
		scenario     string
		values       []string
		expectedSort []string
	}{
		{
			scenario:     "UUIDs with digit suffixes",
			values:       []string{"2521a33d-1fb4-4279-80fe-8a945285b8f4_12.pdf", "2521a33d-1fb4-4279-80fe-8a945285b8f4_1.pdf", "2521a33d-1fb4-4279-80fe-8a945285b8f4_10.pdf", "2521a33d-1fb4-4279-80fe-8a945285b8f4_3.pdf"},
			expectedSort: []string{"2521a33d-1fb4-4279-80fe-8a945285b8f4_1.pdf", "2521a33d-1fb4-4279-80fe-8a945285b8f4_3.pdf", "2521a33d-1fb4-4279-80fe-8a945285b8f4_10.pdf", "2521a33d-1fb4-4279-80fe-8a945285b8f4_12.pdf"},
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			sort.Sort(digitSuffixSort(tc.values))

			if !reflect.DeepEqual(tc.values, tc.expectedSort) {
				t.Fatalf("expected %+v but got: %+v", tc.expectedSort, tc.values)
			}
		})
	}
}
