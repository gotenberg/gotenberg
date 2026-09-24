package pdfcpu

import (
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

type digitSuffixSort []string

func (s digitSuffixSort) Len() int {
	return len(s)
}

func (s digitSuffixSort) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s digitSuffixSort) Less(i, j int) bool {
	numI, restI := extractNumber(s[i])
	numJ, restJ := extractNumber(s[j])

	// If both strings contain a number, compare them numerically.
	if numI != -1 && numJ != -1 {
		if numI != numJ {
			return numI < numJ
		}
		// If the numbers are equal, compare the "rest" strings.
		return restI < restJ
	}

	// If one contains a number and the other doesn't, the one with the number
	// comes first.
	if numI != -1 {
		return true
	}
	if numJ != -1 {
		return false
	}

	// Neither has a number; fall back to lexicographical order.
	return s[i] < s[j]
}

func extractNumber(str string) (int, string) {
	str = filepath.Base(str)

	// Check for a number immediately before an extension.
	if matches := extensionSuffixRegexp.FindStringSubmatch(str); len(matches) > 3 {
		if num, err := strconv.Atoi(matches[2]); err == nil {
			// Remove the numeric block but keep the extension.
			return num, matches[1] + matches[3]
		}
	}

	// No numeric portion found.
	return -1, str
}

// Regular expressions used by extractNumber.
var (
	// Matches a numeric block immediately before a file extension.
	extensionSuffixRegexp = regexp.MustCompile(`^(.*?)(\d+)(\.[^.]+)$`)
)

// Interface guard.
var _ sort.Interface = (*digitSuffixSort)(nil)
