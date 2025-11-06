package gotenberg

import (
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

type numberLoc int

const (
	numberNone numberLoc = iota
	numberPrefix
	numberExtSuffix // number right before extension.
	numberSuffix    // trailing number with no extension.
)

// AlphanumericSort implements sort.Interface and helps to sort strings
// alphanumerically by either a numeric prefix or, if missing, a numeric
// suffix.
//
// See:
// https://github.com/gotenberg/gotenberg/issues/805.
// https://github.com/gotenberg/gotenberg/issues/1287.
type AlphanumericSort []string

func (s AlphanumericSort) Len() int {
	return len(s)
}

func (s AlphanumericSort) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s AlphanumericSort) Less(i, j int) bool {
	numI, restI, locI := extractNumber(s[i])
	numJ, restJ, locJ := extractNumber(s[j])

	// Both have a number.
	if numI != -1 && numJ != -1 {
		// Both prefix numbers: numeric first, then rest.
		if locI == numberPrefix && locJ == numberPrefix {
			if numI != numJ {
				return numI < numJ
			}
			return restI < restJ
		}

		// Both are suffix-ish (right-before-ext or trailing): rest first, then
		// number.
		if locI != numberPrefix && locJ != numberPrefix {
			if restI != restJ {
				return restI < restJ
			}
			if numI != numJ {
				return numI < numJ
			}
			return s[i] < s[j]
		}

		// Mixed: one prefix, one not.
		if restI != restJ {
			return restI < restJ
		}
		return locI == numberPrefix
	}

	// One has a number: it comes first.
	if numI != -1 {
		return true
	}
	if numJ != -1 {
		return false
	}

	// Neither has a number: plain lexicographic.
	return s[i] < s[j]
}

func extractNumber(str string) (int, string, numberLoc) {
	str = filepath.Base(str)

	if matches := prefixRegexp.FindStringSubmatch(str); len(matches) > 2 {
		if num, err := strconv.Atoi(matches[1]); err == nil {
			return num, matches[2], numberPrefix
		}
	}
	if matches := extensionSuffixRegexp.FindStringSubmatch(str); len(matches) > 3 {
		if num, err := strconv.Atoi(matches[2]); err == nil {
			return num, matches[1] + matches[3], numberExtSuffix
		}
	}
	if matches := suffixRegexp.FindStringSubmatch(str); len(matches) > 2 {
		if num, err := strconv.Atoi(matches[2]); err == nil {
			return num, matches[1], numberSuffix
		}
	}
	return -1, str, numberNone
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
