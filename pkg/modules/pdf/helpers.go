package pdf

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	litho "github.com/gotenberg/litho/pkg/pdf"
)

// stampFonts maps the 14 standard PDF font names to litho's stamp fonts.
var stampFonts = map[string]litho.StampFont{
	"Helvetica":             litho.FontHelvetica,
	"Helvetica-Bold":        litho.FontHelveticaBold,
	"Helvetica-Oblique":     litho.FontHelveticaOblique,
	"Helvetica-BoldOblique": litho.FontHelveticaBoldOblique,
	"Times-Roman":           litho.FontTimesRoman,
	"Times-Bold":            litho.FontTimesBold,
	"Times-Italic":          litho.FontTimesItalic,
	"Times-BoldItalic":      litho.FontTimesBoldItalic,
	"Courier":               litho.FontCourier,
	"Courier-Bold":          litho.FontCourierBold,
	"Courier-Oblique":       litho.FontCourierOblique,
	"Courier-BoldOblique":   litho.FontCourierBoldOblique,
	"Symbol":                litho.FontSymbol,
	"ZapfDingbats":          litho.FontZapfDingbats,
}

// encryptAlgorithms maps the encrypt algorithms to litho's. AES-256 is the
// default.
var encryptAlgorithms = map[string]litho.EncryptAlgorithm{
	"":        litho.EncryptAES256,
	"aes256":  litho.EncryptAES256,
	"aes128":  litho.EncryptAES128,
	"rc4-128": litho.EncryptRC4_128,
	"rc4-40":  litho.EncryptRC4_40,
}

// requirePages checks a mandatory page range specification is present. Its
// syntax is checked against the document's page count when the stage runs.
func requirePages(pages string) error {
	if strings.TrimSpace(pages) == "" {
		return errors.New(`"pages" must not be empty, for example "1-3,5", "odd" or "last"`)
	}

	return nil
}

// validateOpacity checks an opacity is within 0 (opaque, the default) and 1.
func validateOpacity(opacity float64) error {
	if opacity < 0 || opacity > 1 {
		return fmt.Errorf(`"opacity" must be between 0 and 1, got %g`, opacity)
	}

	return nil
}

// parseColor parses a "#rrggbb" color into RGB components between 0 and 1.
func parseColor(color string) ([3]float64, error) {
	hex := strings.TrimPrefix(color, "#")
	if len(hex) != 6 || !strings.HasPrefix(color, "#") {
		return [3]float64{}, fmt.Errorf(`"color" must be a #rrggbb hexadecimal color, got %q`, color)
	}

	var rgb [3]float64
	for i := range rgb {
		component, err := strconv.ParseUint(hex[i*2:i*2+2], 16, 8)
		if err != nil {
			return [3]float64{}, fmt.Errorf(`"color" must be a #rrggbb hexadecimal color, got %q`, color)
		}

		rgb[i] = float64(component) / 255
	}

	return rgb, nil
}

// imageMimeType returns the MIME type of a stamp image from its extension.
func imageMimeType(filename string) (string, error) {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".png":
		return "image/png", nil
	case ".jpg", ".jpeg":
		return "image/jpeg", nil
	default:
		return "", fmt.Errorf(`"image" must be a .png, .jpg or .jpeg file, got %q`, filename)
	}
}

// validateHttpUrl checks an option holds an absolute http or https URL.
func validateHttpUrl(option, raw string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("%q must be an http:// or https:// URL", option)
	}

	return nil
}

// sortedKeys returns the keys of m in lexical order, for error messages.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	slices.Sort(keys)

	return keys
}

// boolOr returns *b, or fallback when b is nil.
func boolOr(b *bool, fallback bool) bool {
	if b == nil {
		return fallback
	}

	return *b
}
