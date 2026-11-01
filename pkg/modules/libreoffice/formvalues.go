package libreoffice

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
	libreofficeapi "github.com/gotenberg/gotenberg/v9/pkg/modules/libreoffice/api"
	"github.com/gotenberg/gotenberg/v9/pkg/modules/pdf"
)

// The form fields take the values the /forms/pdf stages take for the same
// feature, such as a #rrggbb color or an angle in degrees, and the names the
// PDF specification gives the viewer preferences. LibreOffice wants its own
// encodings: these functions convert to them. Their errors state what the
// field wants, as [api.FormData.Custom] reports them.

// errWantColor happens when a color form field is not a #rrggbb color.
var errWantColor = errors.New("want a #rrggbb color")

// parseColor converts a #rrggbb color to the decimal RGB value LibreOffice
// reads, such as 8388223 for #7ffe7f.
func parseColor(value string) (int, error) {
	hex, ok := strings.CutPrefix(value, "#")
	if !ok || len(hex) != 6 {
		return 0, errWantColor
	}

	rgb, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return 0, errWantColor
	}

	return int(rgb), nil
}

// parseRotation converts an angle in degrees, counter-clockwise, to the
// tenths of a degree LibreOffice reads.
func parseRotation(value string) (int, error) {
	degrees, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(degrees) || math.IsInf(degrees, 0) {
		return 0, errors.New("want an angle in degrees")
	}

	return int(math.Round(math.Mod(degrees, 360) * 10)), nil
}

// enum maps the names of a form field's values to the numbers LibreOffice
// reads, in the order of those numbers.
type enum []string

func (e enum) parse(value string) (int, error) {
	i := slices.Index(e, value)
	if i < 0 {
		return 0, fmt.Errorf("want %s", api.OrList(e))
	}

	return i, nil
}

// The viewer preferences, named after the PDF specification's /PageMode and
// /PageLayout values (ISO 32000-1, Table 28). The order gives LibreOffice's
// numbers.
var (
	initialViews   = enum{"default", "outline", "thumbnails"}
	magnifications = enum{"default", "fitWindow", "fitWidth", "fitVisible", "zoom"}
	// LibreOffice's PageLayout is 0 for the viewer's default, 1 for
	// /SinglePage, 2 for /OneColumn and 3 for /TwoColumnRight. It writes no
	// /TwoColumnLeft, and discards its FirstPageOnLeft setting, so the two
	// column layouts always open with the first page on the right (see
	// filter/source/pdf/pdfexport.cxx).
	pageLayouts = enum{"default", "singlePage", "oneColumn", "twoColumnRight"}
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

// intoPointer returns a [api.FormData.Custom] binding that parses a value into
// a new int target points to, and keeps target nil when the field is empty, so
// that a value the client sets, 0 included, tells from none.
func intoPointer(target **int, parse func(string) (int, error)) func(string) error {
	return func(value string) error {
		if value == "" {
			return nil
		}

		n, err := parse(value)
		if err != nil {
			return err
		}

		*target = &n

		return nil
	}
}

// renamedHint is the fix of a Gotenberg 8 field that Gotenberg 9 renamed.
func renamedHint(field string) string {
	return fmt.Sprintf("Gotenberg 9 renamed it '%s'", field)
}

// gotenberg8Fields maps the Gotenberg 8 LibreOffice fields that Gotenberg 9
// renamed or removed to their fix. See the upgrade guide.
var gotenberg8Fields = map[string]string{
	"nativePageRanges":           renamedHint("nativePages"),
	"exportBookmarks":            renamedHint("nativeBookmarks"),
	"generateTaggedPdf":          renamedHint("nativeTaggedPdf"),
	"singlePageSheets":           renamedHint("singlePage"),
	"quality":                    renamedHint("nativeQuality"),
	"losslessImageCompression":   renamedHint("nativeLosslessImageCompression"),
	"reduceImageResolution":      renamedHint("nativeReduceImageResolution"),
	"maxImageResolution":         renamedHint("nativeMaxImageResolution"),
	"nativeWatermarkFontHeight":  renamedHint("nativeWatermarkFontSize"),
	"nativeWatermarkFontName":    renamedHint("nativeWatermarkFont"),
	"nativeWatermarkRotateAngle": renamedHint("nativeWatermarkRotation") + ", in degrees",
	"pdfua":                      renamedHint("nativePdfua") + ", which takes the string " + pdfuaProfile,
	"firstPageOnLeft":            "Gotenberg 9 removed it: LibreOffice ignores it",
	"nativePdfFormats":           "Gotenberg 9 removed it: 'nativePdfa' and 'nativePdfua' always apply natively",
	"optimizeImages":             imageQualityHint,
	"imageQuality":               imageQualityHint,
}

// imageQualityHint is the fix of the Gotenberg 8 fields that lowered image
// quality: LibreOffice does at export, while the optimize stage is lossless.
const imageQualityHint = `Gotenberg 9 lowers image quality at export: use 'nativeQuality' or 'nativeReduceImageResolution'; {"op":"optimize"} is lossless`

// pdfaHint is the fix of the Gotenberg 8 'pdfa' field, given the profile it
// sent: 'nativePdfa' writes four PDF/A profiles, a remediate stage the others.
func pdfaHint(profile string) string {
	stage := pdf.PdfaRemediateStage(profile)
	native := libreofficeapi.PdfFormats{PdfA: profile}

	if profile != "" && native.Validate() == nil {
		return fmt.Sprintf("%s, or add %s to 'stages'", renamedHint("nativePdfa"), stage)
	}

	return fmt.Sprintf("%s, which takes PDF/A-1b, PDF/A-2b, PDF/A-3b or PDF/A-4, or add %s to 'stages'", renamedHint("nativePdfa"), stage)
}

// formFieldHint is the [api.FormData.RejectUnknown] hint of the LibreOffice
// route: a renamed or removed field, or the stage that replaced a
// post-processing field.
func formFieldHint(field, value string, file bool) string {
	if !file {
		if field == "pdfa" {
			return pdfaHint(value)
		}

		if hint, ok := gotenberg8Fields[field]; ok {
			return hint
		}
	}

	return pdf.FormFieldHint(field, value, file)
}
