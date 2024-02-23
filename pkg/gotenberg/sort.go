package gotenberg

import (
	"regexp"
	"sort"
	"strconv"
)

// AlphanumericSort implements sort.Interface and helps to sort strings
// alphanumerically.
//
// See https://github.com/gotenberg/gotenberg/issues/805.
type AlphanumericSort []string

func (s AlphanumericSort) Len() int {
	return len(s)
}

func (s AlphanumericSort) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s AlphanumericSort) Less(i, j int) bool {
	numI, restI := extractPrefix(s[i])
	numJ, restJ := extractPrefix(s[j])

	// Compares numerical prefixes if they exist.
	if numI != -1 && numJ != -1 {
		if numI != numJ {
			return numI < numJ
		}
		// If numbers are equal, falls back to string comparison of the rest.
		return restI < restJ
	}

	// If one has a numerical prefix and the other doesn't, the one with the
	// number comes first.
	if numI != -1 {
		return true
	}
	if numJ != -1 {
		return false
	}

	// If neither has a numerical prefix, compare as strings
	return s[i] < s[j]
}

// extractPrefix attempts to extract a numerical prefix and the rest of the filename
func extractPrefix(filename string) (int, string) {
	matches := numPrefixRegexp.FindStringSubmatch(filename)
	if len(matches) > 2 {
		prefix, err := strconv.Atoi(matches[1])
		if err == nil {
			return prefix, matches[2]
		}
	}

	// Returns -1 if no numerical prefix is found, indicating to just compare
	// as strings.
	return -1, filename
}

var numPrefixRegexp = regexp.MustCompile(`^(\d+)(.*)$`)

// Interface guard.
var (
	_ sort.Interface = (*AlphanumericSort)(nil)
)
