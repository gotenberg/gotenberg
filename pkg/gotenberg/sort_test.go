package gotenberg

import (
	"reflect"
	"sort"
	"testing"
)

func TestAlphanumericSort(t *testing.T) {
	for _, tc := range []struct {
		scenario     string
		values       []string
		expectedSort []string
	}{
		{
			scenario:     "numeric and letters",
			values:       []string{"10qux.pdf", "2_baz.txt", "2_aza.txt", "1bar.pdf", "Afoo.txt", "Bbar.docx", "25zeta.txt", "3.pdf", "4_foo.pdf"},
			expectedSort: []string{"1bar.pdf", "2_aza.txt", "2_baz.txt", "3.pdf", "4_foo.pdf", "10qux.pdf", "25zeta.txt", "Afoo.txt", "Bbar.docx"},
		},
		{
			scenario:     "hrtime (PHP library)",
			values:       []string{"245654773395259", "245654773395039", "245654773395149", "245654773394919", "245654773394369"},
			expectedSort: []string{"245654773394369", "245654773394919", "245654773395039", "245654773395149", "245654773395259"},
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			sort.Sort(AlphanumericSort(tc.values))

			if !reflect.DeepEqual(tc.values, tc.expectedSort) {
				t.Fatalf("expected %+v but got: %+v", tc.expectedSort, tc.values)
			}
		})
	}
}
