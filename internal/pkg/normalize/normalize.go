package normalize

import (
	"unicode"

	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// String normalizes given string.
func String(str string) (string, error) {
	const op string = "normalize.String"
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, err := transform.String(t, str)
	if err != nil {
		return "", xerror.New(op, err)
	}
	return result, nil
}
