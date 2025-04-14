package gotenberg

import (
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

// AlphanumericSort implements sort.Interface and helps to sort strings
// alphanumerically by either a numeric prefix or, if missing, a numeric
// suffix.
//
// See: https://github.com/gotenberg/gotenberg/issues/805.
type AlphanumericSort []string

func (s AlphanumericSort) Len() int {
	return len(s)
}

func (s AlphanumericSort) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s AlphanumericSort) Less(i, j int) bool {
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

// extractNumber attempts to extract a numeric portion from the filename.
// It first checks for a numeric prefix (digits at the beginning).
// If none is found, it next attempts to match a number immediately before the
// extension (for filenames such as "sample1_1.pdf").
// If that fails, it then attempts a trailing numeric pattern.
// If no number is found, it returns -1 and the original string.
func extractNumber(str string) (int, string) {
	// See https://github.com/gotenberg/gotenberg/issues/1168.
	str = filepath.Base(str)

	// Check for a numeric prefix.
	if matches := prefixRegexp.FindStringSubmatch(str); len(matches) > 2 {
		if num, err := strconv.Atoi(matches[1]); err == nil {
			return num, matches[2]
		}
	}

	// Check for a number immediately before an extension.
	if matches := extensionSuffixRegexp.FindStringSubmatch(str); len(matches) > 3 {
		if num, err := strconv.Atoi(matches[2]); err == nil {
			// Remove the numeric block but keep the extension.
			return num, matches[1] + matches[3]
		}
	}

	// Check for a trailing number (with no extension following).
	if matches := suffixRegexp.FindStringSubmatch(str); len(matches) > 2 {
		if num, err := strconv.Atoi(matches[2]); err == nil {
			return num, matches[1]
		}
	}

	// No numeric portion found.
	return -1, str
}

// Regular expressions used by extractNumber.
var (
	// Matches a numeric prefix: one or more digits at the start.
	prefixRegexp = regexp.MustCompile(`^(\d+)(.*)$`)
	// Matches a numeric block immediately before a file extension.
	extensionSuffixRegexp = regexp.MustCompile(`^(.*?)(\d+)(\.[^.]+)$`)
	// Matches a trailing numeric sequence when there is no extension.
	suffixRegexp = regexp.MustCompile(`^(.*?)(\d+)$`)
)

// Interface guard.
var _ sort.Interface = (*AlphanumericSort)(nil)
