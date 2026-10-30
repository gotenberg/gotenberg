package libreoffice

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
)

// The form fields take the values the /forms/pdf stages take for the same
// feature, such as a #rrggbb color or an angle in degrees, and the names the
// PDF specification gives the viewer preferences. LibreOffice wants its own
// encodings: these functions convert to them.

// parseColor converts a #rrggbb color to the decimal RGB value LibreOffice
// reads, such as 8388223 for #7ffe7f.
func parseColor(value string) (int, error) {
	hex, ok := strings.CutPrefix(value, "#")
	if !ok || len(hex) != 6 {
		return 0, fmt.Errorf("must be a #rrggbb hexadecimal color, got %q", value)
	}

	rgb, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return 0, fmt.Errorf("must be a #rrggbb hexadecimal color, got %q", value)
	}

	return int(rgb), nil
}

// parseRotation converts an angle in degrees, counter-clockwise, to the
// tenths of a degree LibreOffice reads.
func parseRotation(value string) (int, error) {
	degrees, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(degrees) || math.IsInf(degrees, 0) {
		return 0, fmt.Errorf("must be an angle in degrees, got %q", value)
	}

	return int(math.Round(math.Mod(degrees, 360) * 10)), nil
}

// enum maps the names of a form field's values to the numbers LibreOffice
// reads, in the order of those numbers.
type enum []string

func (e enum) parse(value string) (int, error) {
	i := slices.Index(e, value)
	if i < 0 {
		return 0, fmt.Errorf("must be one of %s, got %q", strings.Join(e, ", "), value)
	}

	return i, nil
}

// The viewer preferences, named after the PDF specification's /PageMode and
// /PageLayout values (ISO 32000-1, Table 28). The order gives LibreOffice's
// numbers.
var (
	initialViews   = enum{"default", "outline", "thumbnails"}
	magnifications = enum{"default", "fitWindow", "fitWidth", "fitVisible", "zoom"}
	pageLayouts    = enum{"default", "singlePage", "oneColumn", "twoColumnLeft", "twoColumnRight"}
)

// pdfuaProfile is the only PDF/UA profile LibreOffice writes.
const pdfuaProfile = "PDF/UA-1"

// into returns a [api.FormData.Custom] binding that parses a value into
// target, and keeps target's default when the field is empty.
func into(target *int, parse func(string) (int, error)) func(string) error {
	return func(value string) error {
		if value == "" {
			return nil
		}

		n, err := parse(value)
		if err != nil {
			return err
		}

		*target = n

		return nil
	}
}
