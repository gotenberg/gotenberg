package pdfengines

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

// FormDataPdfSplitMode creates a [gotenberg.SplitMode] from the form data.
func FormDataPdfSplitMode(form *api.FormData, mandatory bool) gotenberg.SplitMode {
	var (
		mode  string
		span  string
		unify bool
	)

	splitModeFunc := func(value string) error {
		if value != "" && value != gotenberg.SplitModeIntervals && value != gotenberg.SplitModePages {
			return fmt.Errorf("wrong value, expected either '%s' or '%s'", gotenberg.SplitModeIntervals, gotenberg.SplitModePages)
		}
		mode = value
		return nil
	}

	splitSpanFunc := func(value string) error {
		value = strings.Join(strings.Fields(value), "")

		if mode == gotenberg.SplitModeIntervals {
			intValue, err := strconv.Atoi(value)
			if err != nil {
				return err
			}
			if intValue < 1 {
				return errors.New("value is inferior to 1")
			}
		}

		span = value

		return nil
	}

	if mandatory {
		form.
			MandatoryCustom("splitMode", func(value string) error {
				return splitModeFunc(value)
			}).
			MandatoryCustom("splitSpan", func(value string) error {
				return splitSpanFunc(value)
			})
	} else {
		form.
			Custom("splitMode", func(value string) error {
				return splitModeFunc(value)
			}).
			Custom("splitSpan", func(value string) error {
				return splitSpanFunc(value)
			})
	}

	form.
		Bool("splitUnify", &unify, false).
		Custom("splitUnify", func(value string) error {
			if value != "" && unify && mode != gotenberg.SplitModePages {
				return fmt.Errorf("unify is not available for split mode '%s'", mode)
			}
			return nil
		})

	return gotenberg.SplitMode{
		Mode:  mode,
		Span:  span,
		Unify: unify,
	}
}

// FormDataPdfFormats creates [gotenberg.PdfFormats] from the form data.
// Fallback to the default value if the considered key is not present.
func FormDataPdfFormats(form *api.FormData) gotenberg.PdfFormats {
	var (
		pdfa  string
		pdfua bool
	)

	form.
		String("pdfa", &pdfa, "").
		Bool("pdfua", &pdfua, false)

	return gotenberg.PdfFormats{
		PdfA:  pdfa,
		PdfUa: pdfua,
	}
}

// FormDataPdfMetadata creates metadata object from the form data.
func FormDataPdfMetadata(form *api.FormData, mandatory bool) map[string]any {
	var metadata map[string]any

	metadataFunc := func(value string) error {
		if len(value) > 0 {
			err := json.Unmarshal([]byte(value), &metadata)
			if err != nil {
				return fmt.Errorf("unmarshal metadata: %w", err)
			}
		}
		return nil
	}

	if mandatory {
		form.MandatoryCustom("metadata", func(value string) error {
			return metadataFunc(value)
		})
	} else {
		form.Custom("metadata", func(value string) error {
			return metadataFunc(value)
		})
	}

	return metadata
}

// FormDataPdfBookmarks creates bookmarks from the form data.
func FormDataPdfBookmarks(form *api.FormData, mandatory bool) any {
	var bookmarks any

	bookmarksFunc := func(value string) error {
		if len(value) > 0 {
			var list []gotenberg.Bookmark
			err := json.Unmarshal([]byte(value), &list)
			if err == nil {
				bookmarks = list
				return nil
			}

			var m map[string][]gotenberg.Bookmark
			err = json.Unmarshal([]byte(value), &m)
			if err == nil {
				bookmarks = m
				return nil
			}

			return fmt.Errorf("unmarshal bookmarks: %w", err)
		}
		return nil
	}

	if mandatory {
		form.MandatoryCustom("bookmarks", func(value string) error {
			return bookmarksFunc(value)
		})
	} else {
		form.Custom("bookmarks", func(value string) error {
			return bookmarksFunc(value)
		})
	}

	return bookmarks
}

// FormDataPdfRotate creates rotation parameters from the form data.
func FormDataPdfRotate(form *api.FormData, mandatory bool) (int, string) {
	var angle int
	var pages string

	angleFunc := func(value string) error {
		if value == "" {
			return nil
		}
		v, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		if v != 90 && v != 180 && v != 270 {
			return errors.New("wrong value, expected 90, 180, or 270")
		}
		angle = v
		return nil
	}

	if mandatory {
		form.MandatoryCustom("rotateAngle", func(value string) error {
			return angleFunc(value)
		})
	} else {
		form.Custom("rotateAngle", func(value string) error {
			return angleFunc(value)
		})
	}
	form.String("rotatePages", &pages, "")

	return angle, pages
}

// RotateStub rotates pages of PDF files. If angle is 0, it does nothing.
func RotateStub(ctx *api.Context, engine gotenberg.PdfEngine, angle int, pages string, inputPaths []string) error {
	if angle == 0 {
		return nil
	}

	for _, inputPath := range inputPaths {
		err := engine.Rotate(ctx, ctx.Log(), inputPath, angle, pages)
		if err != nil {
			return fmt.Errorf("rotate '%s': %w", inputPath, err)
		}
	}

	return nil
}

// ValidatePdfFormatsCompat checks for incompatible combinations of PDF formats
// with other features and returns an appropriate error if found.
func ValidatePdfFormatsCompat(pdfFormats gotenberg.PdfFormats, userPassword string, embedPaths []string) error {
	zeroValued := gotenberg.PdfFormats{}
	if pdfFormats == zeroValued {
		return nil
	}

	// PDF/A forbids encryption per the standard.
	if pdfFormats.PdfA != "" && userPassword != "" {
		return api.WrapError(
			errors.New("PDF/A format is incompatible with encryption"),
			api.NewSentinelHttpError(
				http.StatusBadRequest,
				"Invalid form data: PDF/A format is incompatible with encryption",
			),
		)
	}

	// Only PDF/A-3 variants allow embedded file attachments.
	if pdfFormats.PdfA != "" && len(embedPaths) > 0 {
		if pdfFormats.PdfA != gotenberg.PdfA3a && pdfFormats.PdfA != gotenberg.PdfA3b && pdfFormats.PdfA != gotenberg.PdfA3u {
			return api.WrapError(
				fmt.Errorf("PDF format '%s' does not support embedded files", pdfFormats.PdfA),
				api.NewSentinelHttpError(
					http.StatusBadRequest,
					fmt.Sprintf("Invalid form data: PDF format '%s' does not support embedded files; only PDF/A-3 variants allow attachments", pdfFormats.PdfA),
				),
			)
		}
	}

	return nil
}

// MergeStub merges given PDFs. If only one input PDF, it does nothing and
// returns the corresponding input path.
func MergeStub(ctx *api.Context, engine gotenberg.PdfEngine, inputPaths []string) (string, error) {
	if len(inputPaths) == 0 {
		return "", errors.New("no input paths")
	}

	if len(inputPaths) == 1 {
		return inputPaths[0], nil
	}

	outputPath := ctx.GeneratePath(".pdf")
	err := engine.Merge(ctx, ctx.Log(), inputPaths, outputPath)
	if err != nil {
		return "", fmt.Errorf("merge %d PDFs: %w", len(inputPaths), err)
	}

	return outputPath, nil
}

// SplitPdfStub splits a list of PDF files based on [gotenberg.SplitMode].
// It returns a list of output paths or the list of provided input paths if no
// split requested.
func SplitPdfStub(ctx *api.Context, engine gotenberg.PdfEngine, mode gotenberg.SplitMode, inputPaths []string) ([]string, error) {
	zeroValued := gotenberg.SplitMode{}
	if mode == zeroValued {
		return inputPaths, nil
	}

	var outputPaths []string
	for _, inputPath := range inputPaths {
		originalName := ctx.OriginalFilename(inputPath)
		originalNameNoExt := strings.TrimSuffix(originalName, filepath.Ext(originalName))
		outputDirPath, err := ctx.CreateSubDirectory(uuid.New().String())
		if err != nil {
			return nil, fmt.Errorf("create subdirectory from input path: %w", err)
		}

		paths, err := engine.Split(ctx, ctx.Log(), mode, inputPath, outputDirPath)
		if err != nil {
			return nil, fmt.Errorf("split PDF '%s': %w", inputPath, err)
		}

		// Keep the original filename.
		for i, path := range paths {
			var newPath string
			var newOriginal string
			if mode.Unify && mode.Mode == gotenberg.SplitModePages {
				newOriginal = fmt.Sprintf("%s.pdf", originalNameNoExt)
				newPath = fmt.Sprintf(
					"%s/%s",
					outputDirPath, uuid.New().String()+".pdf",
				)
			} else {
				newOriginal = fmt.Sprintf("%s_%d.pdf", originalNameNoExt, i)
				newPath = fmt.Sprintf(
					"%s/%s",
					outputDirPath, uuid.New().String()+".pdf",
				)
			}

			err = ctx.Rename(path, newPath)
			if err != nil {
				return nil, fmt.Errorf("rename path: %w", err)
			}

			ctx.RegisterDiskPath(newPath, newOriginal)
			outputPaths = append(outputPaths, newPath)

			if mode.Unify && mode.Mode == gotenberg.SplitModePages {
				break
			}
		}
	}

	return outputPaths, nil
}

// FlattenStub merges annotation appearances with page content for each given
// PDF, effectively deleting the original annotations.
func FlattenStub(ctx *api.Context, engine gotenberg.PdfEngine, inputPaths []string) error {
	for _, inputPath := range inputPaths {
		err := engine.Flatten(ctx, ctx.Log(), inputPath)
		if err != nil {
			return fmt.Errorf("flatten '%s': %w", inputPath, err)
		}
	}

	return nil
}

// defaultImageQuality is the JPEG quality applied by the image optimization
// feature when the imageQuality form field is not set.
const defaultImageQuality = 80

// FormDataPdfOptimize extracts the image-optimization options from the form
// data: whether to optimize the images, and the JPEG quality (1 to 100) to
// apply to each re-encoded image.
func FormDataPdfOptimize(form *api.FormData) (bool, int) {
	var (
		optimizeImages bool
		imageQuality   int
	)

	form.
		Bool("optimizeImages", &optimizeImages, false).
		Custom("imageQuality", func(value string) error {
			if value == "" {
				imageQuality = defaultImageQuality
				return nil
			}

			intValue, err := strconv.Atoi(value)
			if err != nil {
				return err
			}

			if intValue < 1 {
				return errors.New("value is inferior to 1")
			}

			if intValue > 100 {
				return errors.New("value is superior to 100")
			}

			imageQuality = intValue
			return nil
		})

	return optimizeImages, imageQuality
}

// OptimizeStub re-encodes the images of each given PDF to shrink the file when
// optimizeImages is set, leaving text, vectors and structure untouched. It does
// nothing when optimizeImages is false.
func OptimizeStub(ctx *api.Context, engine gotenberg.PdfEngine, optimizeImages bool, imageQuality int, inputPaths []string) error {
	if !optimizeImages {
		return nil
	}

	for _, inputPath := range inputPaths {
		err := engine.OptimizeImages(ctx, ctx.Log(), imageQuality, inputPath)
		if err != nil {
			return fmt.Errorf("optimize images of '%s': %w", inputPath, err)
		}
	}

	return nil
}

// ConvertStub transforms a given PDF to the specified formats defined in
// [gotenberg.PdfFormats]. If no format, it does nothing and returns the input
// paths.
func ConvertStub(ctx *api.Context, engine gotenberg.PdfEngine, formats gotenberg.PdfFormats, inputPaths []string) ([]string, error) {
	zeroValued := gotenberg.PdfFormats{}
	if formats == zeroValued {
		return inputPaths, nil
	}

	outputPaths := make([]string, len(inputPaths))
	for i, inputPath := range inputPaths {
		outputPaths[i] = ctx.GeneratePath(".pdf")

		err := engine.Convert(ctx, ctx.Log(), formats, inputPath, outputPaths[i])
		if err != nil {
			return nil, fmt.Errorf("convert '%s': %w", inputPath, err)
		}
	}

	return outputPaths, nil
}

// WriteMetadataStub writes the metadata into PDF files. If no metadata, it
// does nothing.
func WriteMetadataStub(ctx *api.Context, engine gotenberg.PdfEngine, metadata map[string]any, inputPaths []string) error {
	if len(metadata) == 0 {
		return nil
	}

	for _, inputPath := range inputPaths {
		err := engine.WriteMetadata(ctx, ctx.Log(), metadata, inputPath)
		if err != nil {
			return fmt.Errorf("write metadata into '%s': %w", inputPath, err)
		}
	}

	return nil
}

// documentTitle returns the input PDF's Title metadata entry, falling back to
// the original filename without its extension when the entry is absent, blank,
// or cannot be read. It labels the per-document entries the merge route's
// titleBookmarks feature generates.
func documentTitle(ctx *api.Context, engine gotenberg.PdfEngine, inputPath, filename string) string {
	fallback := strings.TrimSuffix(filename, filepath.Ext(filename))

	metadata, err := engine.ReadMetadata(ctx, ctx.Log(), inputPath)
	if err != nil {
		ctx.Log().WarnContext(ctx, fmt.Sprintf("read metadata of '%s' for title bookmark, using filename: %s", filename, err))
		return fallback
	}

	title, ok := metadata["Title"].(string)
	if !ok || strings.TrimSpace(title) == "" {
		return fallback
	}

	return title
}

func shiftBookmarks(bookmarks []gotenberg.Bookmark, offset int) []gotenberg.Bookmark {
	if offset == 0 {
		return bookmarks
	}
	shifted := make([]gotenberg.Bookmark, len(bookmarks))
	for i, b := range bookmarks {
		shifted[i] = gotenberg.Bookmark{
			Title:    b.Title,
			Page:     b.Page + offset,
			Children: shiftBookmarks(b.Children, offset),
		}
	}
	return shifted
}

// WriteBookmarksStub writes the bookmarks into PDF files. If no bookmarks, it
// does nothing.
func WriteBookmarksStub(ctx *api.Context, engine gotenberg.PdfEngine, bookmarks any, inputPaths []string) error {
	if bookmarks == nil {
		return nil
	}

	switch b := bookmarks.(type) {
	case []gotenberg.Bookmark:
		if len(b) == 0 {
			return nil
		}

		for _, inputPath := range inputPaths {
			err := engine.WriteBookmarks(ctx, ctx.Log(), inputPath, b)
			if err != nil {
				return fmt.Errorf("write bookmarks into '%s': %w", inputPath, err)
			}
		}
	case map[string][]gotenberg.Bookmark:
		for _, inputPath := range inputPaths {
			filename := ctx.OriginalFilename(inputPath)
			if specificBookmarks, ok := b[filename]; ok {
				err := engine.WriteBookmarks(ctx, ctx.Log(), inputPath, specificBookmarks)
				if err != nil {
					return fmt.Errorf("write bookmarks into '%s': %w", inputPath, err)
				}
			}
		}
	default:
		// Should not happen.
		return fmt.Errorf("bookmarks type '%T' not supported", bookmarks)
	}

	return nil
}

// FormDataPdfEmbeds extracts embedded file paths from form data.
// Only files uploaded with the "embeds" field name are included.
func FormDataPdfEmbeds(form *api.FormData) []string {
	var embedPaths []string
	form.Embeds(&embedPaths)
	return embedPaths
}

// FormDataPdfEmbedsMetadata extracts embeds metadata from form data.
// The "embedsMetadata" field is a JSON string keyed by filename.
func FormDataPdfEmbedsMetadata(form *api.FormData) map[string]map[string]string {
	var metadata map[string]map[string]string
	form.EmbedsMetadata(&metadata)
	return metadata
}

// EmbedFilesMetadataStub sets metadata on embedded files in PDFs.
func EmbedFilesMetadataStub(ctx *api.Context, engine gotenberg.PdfEngine, metadata map[string]map[string]string, inputPaths []string) error {
	if len(metadata) == 0 {
		return nil
	}

	for _, inputPath := range inputPaths {
		err := engine.EmbedFilesMetadata(ctx, ctx.Log(), metadata, inputPath)
		if err != nil {
			return fmt.Errorf("set embeds metadata on PDF '%s': %w", inputPath, err)
		}
	}

	return nil
}

// FormDataPdfFacturX extracts the Factur-X parameters and the invoice XML path
// from form data. Factur-X is requested when both facturxConformanceLevel and
// facturxXml are provided. The embedded XML always takes the canonical
// [gotenberg.FacturXDocumentFileName] name.
func FormDataPdfFacturX(form *api.FormData) (gotenberg.FacturX, string) {
	var (
		facturxXmlPath   string
		conformanceLevel string
		documentType     string
		version          string
	)

	form.
		FacturXXml(&facturxXmlPath).
		Custom("facturxConformanceLevel", func(value string) error {
			conformanceLevel = value
			switch value {
			case "",
				gotenberg.FacturXConformanceMinimum,
				gotenberg.FacturXConformanceBasicWL,
				gotenberg.FacturXConformanceBasic,
				gotenberg.FacturXConformanceEN16931,
				gotenberg.FacturXConformanceExtended,
				gotenberg.FacturXConformanceXRechnung:
				return nil
			default:
				return fmt.Errorf("unsupported conformance level '%s'", value)
			}
		}).
		Custom("facturxDocumentType", func(value string) error {
			if value == "" {
				documentType = gotenberg.FacturXDocumentTypeInvoice
				return nil
			}
			documentType = value
			switch value {
			case gotenberg.FacturXDocumentTypeInvoice,
				gotenberg.FacturXDocumentTypeOrder,
				gotenberg.FacturXDocumentTypeOrderResponse,
				gotenberg.FacturXDocumentTypeOrderChange:
				return nil
			default:
				return fmt.Errorf("unsupported document type '%s'", value)
			}
		}).
		String("facturxVersion", &version, "1.0")

	return gotenberg.FacturX{
		ConformanceLevel: conformanceLevel,
		DocumentType:     documentType,
		DocumentFileName: gotenberg.FacturXDocumentFileName,
		Version:          version,
	}, facturxXmlPath
}

// isPdfA3 reports whether the format is a PDF/A-3 variant, the only family that
// allows the embedded files Factur-X requires.
func isPdfA3(pdfA string) bool {
	return pdfA == gotenberg.PdfA3a || pdfA == gotenberg.PdfA3b || pdfA == gotenberg.PdfA3u
}

// ValidateFacturXCompat enforces the Factur-X pairing and PDF/A-3 rules. It
// returns a 400 error when the request is half-specified, or when an explicit
// PDF/A format is not a PDF/A-3 variant.
func ValidateFacturXCompat(facturX gotenberg.FacturX, facturxXmlPath string, pdfFormats gotenberg.PdfFormats) error {
	if facturX.ConformanceLevel == "" && facturxXmlPath == "" {
		return nil
	}

	if facturX.ConformanceLevel == "" {
		return api.WrapError(
			errors.New("facturxConformanceLevel is required when facturxXml is provided"),
			api.NewSentinelHttpError(http.StatusBadRequest, "Invalid form data: 'facturxConformanceLevel' is required when 'facturxXml' is provided"),
		)
	}

	if facturxXmlPath == "" {
		return api.WrapError(
			errors.New("facturxXml is required when facturxConformanceLevel is set"),
			api.NewSentinelHttpError(http.StatusBadRequest, "Invalid form data: 'facturxXml' file is required when 'facturxConformanceLevel' is set"),
		)
	}

	if pdfFormats.PdfA != "" && !isPdfA3(pdfFormats.PdfA) {
		return api.WrapError(
			fmt.Errorf("Factur-X requires PDF/A-3, got '%s'", pdfFormats.PdfA),
			api.NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("Invalid form data: Factur-X requires a PDF/A-3 variant (PDF/A-3a, PDF/A-3b, or PDF/A-3u), got '%s'", pdfFormats.PdfA)),
		)
	}

	return nil
}

// FacturXPdfFormats returns the PDF/A formats to convert to so the output meets
// Factur-X's PDF/A-3 requirement. It returns pdfFormats unchanged when Factur-X
// is not requested or the caller already asked for a PDF/A-3 variant. Otherwise
// it defaults to PDF/A-3b, except for pre-existing PDFs (sourceDoc false) that
// already carry PDF/A-3, which are left untouched.
func FacturXPdfFormats(ctx *api.Context, engine gotenberg.PdfEngine, facturX gotenberg.FacturX, pdfFormats gotenberg.PdfFormats, sourceDoc bool, inputPaths []string) gotenberg.PdfFormats {
	if facturX.ConformanceLevel == "" || isPdfA3(pdfFormats.PdfA) {
		return pdfFormats
	}

	if sourceDoc {
		pdfFormats.PdfA = gotenberg.PdfA3b
		return pdfFormats
	}

	// Pre-existing PDFs: keep an already-PDF/A-3 input as-is, otherwise default
	// to PDF/A-3b.
	for _, inputPath := range inputPaths {
		part, _, err := engine.ReadPdfAConformance(ctx, ctx.Log(), inputPath)
		if err != nil {
			ctx.Log().DebugContext(ctx, fmt.Sprintf("read PDF/A conformance of '%s', assuming not PDF/A-3: %s", inputPath, err))
			part = ""
		}
		if part != "3" {
			pdfFormats.PdfA = gotenberg.PdfA3b
			return pdfFormats
		}
	}

	return pdfFormats
}

// ApplyFacturXStub turns each input PDF into a Factur-X document: it embeds the
// CII invoice XML under the canonical name with AFRelationship "Alternative",
// then injects the fx XMP metadata. The inputs must already be PDF/A-3 (see
// [FacturXPdfFormats]). It is a no-op when Factur-X is not requested.
func ApplyFacturXStub(ctx *api.Context, engine gotenberg.PdfEngine, facturX gotenberg.FacturX, facturxXmlPath string, inputPaths []string) error {
	if facturX.ConformanceLevel == "" {
		return nil
	}

	err := embedFacturXXml(ctx, engine, facturxXmlPath, inputPaths)
	if err != nil {
		return err
	}

	metadata := map[string]map[string]string{
		facturX.DocumentFileName: {
			"mimeType":     "text/xml",
			"relationship": "Alternative",
		},
	}
	err = EmbedFilesMetadataStub(ctx, engine, metadata, inputPaths)
	if err != nil {
		return fmt.Errorf("set Factur-X embed metadata: %w", err)
	}

	err = InjectFacturXXMPStub(ctx, engine, facturX, inputPaths)
	if err != nil {
		return err
	}

	return nil
}

// embedFacturXXml embeds the Factur-X invoice XML into each PDF under the
// canonical [gotenberg.FacturXDocumentFileName] name, regardless of the
// uploaded file name.
func embedFacturXXml(ctx *api.Context, engine gotenberg.PdfEngine, facturxXmlPath string, inputPaths []string) error {
	embedDir, err := ctx.CreateSubDirectory(uuid.New().String())
	if err != nil {
		return fmt.Errorf("create Factur-X embed subdirectory: %w", err)
	}

	canonicalPath := fmt.Sprintf("%s/%s", embedDir, gotenberg.FacturXDocumentFileName)
	err = os.Symlink(facturxXmlPath, canonicalPath)
	if err != nil {
		return fmt.Errorf("symlink Factur-X invoice XML: %w", err)
	}

	for _, inputPath := range inputPaths {
		err = engine.EmbedFiles(ctx, ctx.Log(), []string{canonicalPath}, inputPath)
		if err != nil {
			return fmt.Errorf("embed Factur-X invoice XML into PDF '%s': %w", inputPath, err)
		}
	}

	return nil
}

// InjectFacturXXMPStub injects Factur-X XMP metadata into PDF files. If the
// Factur-X data is not set, it does nothing.
func InjectFacturXXMPStub(ctx *api.Context, engine gotenberg.PdfEngine, facturX gotenberg.FacturX, inputPaths []string) error {
	if facturX.ConformanceLevel == "" {
		return nil
	}

	for _, inputPath := range inputPaths {
		err := engine.InjectFacturXXMP(ctx, ctx.Log(), facturX, inputPath)
		if err != nil {
			return fmt.Errorf("inject Factur-X XMP into PDF '%s': %w", inputPath, err)
		}
	}

	return nil
}

// FormDataPdfEncrypt extracts the encryption parameters and permissions from
// form data. Permissions default to allowed.
func FormDataPdfEncrypt(form *api.FormData) gotenberg.EncryptOptions {
	var opts gotenberg.EncryptOptions
	form.
		String("userPassword", &opts.UserPassword, "").
		String("ownerPassword", &opts.OwnerPassword, "").
		Bool("allowPrinting", &opts.Permissions.AllowPrinting, true).
		Bool("allowCopying", &opts.Permissions.AllowCopying, true).
		Bool("allowModifying", &opts.Permissions.AllowModifying, true).
		Bool("allowAnnotating", &opts.Permissions.AllowAnnotating, true).
		Bool("allowFillingForms", &opts.Permissions.AllowFillingForms, true).
		Bool("allowAssembling", &opts.Permissions.AllowAssembling, true)
	return opts
}

// ValidatePdfEncryptCompat returns a 400 error when permission restrictions are
// requested without a password to anchor them.
func ValidatePdfEncryptCompat(opts gotenberg.EncryptOptions) error {
	if opts.Permissions.Restricted() && opts.UserPassword == "" && opts.OwnerPassword == "" {
		return api.WrapError(
			errors.New("permission restrictions require a password"),
			api.NewSentinelHttpError(http.StatusBadRequest, "Invalid form data: permission restrictions require a 'userPassword' or 'ownerPassword'"),
		)
	}

	return nil
}

// EncryptPdfStub adds password protection and permission restrictions to PDF
// files. It does nothing when no password is provided.
func EncryptPdfStub(ctx *api.Context, engine gotenberg.PdfEngine, opts gotenberg.EncryptOptions, inputPaths []string) error {
	if opts.UserPassword == "" && opts.OwnerPassword == "" {
		return nil
	}

	for _, inputPath := range inputPaths {
		err := engine.Encrypt(ctx, ctx.Log(), inputPath, opts)
		if err != nil {
			return fmt.Errorf("encrypt PDF '%s': %w", inputPath, err)
		}
	}

	return nil
}

// EmbedFilesStub embeds files into PDF files.
func EmbedFilesStub(ctx *api.Context, engine gotenberg.PdfEngine, embedPaths []string, inputPaths []string) error {
	if len(embedPaths) == 0 {
		return nil
	}

	// Engines like pdfcpu use filepath.Base(path) as the attachment name
	// inside the PDF. Since disk filenames are now UUID-based, we create
	// symlinks with the original names so that embeds are named correctly.
	// See: https://github.com/gotenberg/gotenberg/issues/1500.
	embedDir, err := ctx.CreateSubDirectory(uuid.New().String())
	if err != nil {
		return fmt.Errorf("create embed subdirectory: %w", err)
	}

	resolvedPaths := make([]string, len(embedPaths))
	for i, embedPath := range embedPaths {
		originalName := ctx.OriginalFilename(embedPath)
		resolvedPath := fmt.Sprintf("%s/%s", embedDir, originalName)
		err := os.Symlink(embedPath, resolvedPath)
		if err != nil {
			return fmt.Errorf("symlink embed file '%s': %w", originalName, err)
		}
		resolvedPaths[i] = resolvedPath
	}

	for _, inputPath := range inputPaths {
		err := engine.EmbedFiles(ctx, ctx.Log(), resolvedPaths, inputPath)
		if err != nil {
			return fmt.Errorf("embed files into PDF '%s': %w", inputPath, err)
		}
	}

	return nil
}

// FormDataPdfStamps builds the ordered list of stamps from the repeated stamp
// fields. See [formDataPdfStampsOrWatermarks].
func FormDataPdfStamps(form *api.FormData) ([]gotenberg.Stamp, error) {
	return formDataPdfStampsOrWatermarks(form, "stamp")
}

// FormDataPdfWatermarks builds the ordered list of watermarks from the repeated
// watermark fields. See [formDataPdfStampsOrWatermarks].
func FormDataPdfWatermarks(form *api.FormData) ([]gotenberg.Stamp, error) {
	return formDataPdfStampsOrWatermarks(form, "watermark")
}

// formDataPdfStampsOrWatermarks builds the ordered list of stamps or watermarks
// from the repeated {prefix}Source, {prefix}Expression, {prefix}Pages and
// {prefix}Options fields. The number of entries equals the number of
// {prefix}Source values, so a single occurrence of each field yields one entry,
// preserving the single-stamp/watermark behavior. Fields are aligned by
// position; a missing expression, pages or options entry defaults to empty.
// Image and pdf entries take their file from the uploaded files, in order (see
// [bindStampOrWatermarkFiles]).
func formDataPdfStampsOrWatermarks(form *api.FormData, prefix string) ([]gotenberg.Stamp, error) {
	var sources, expressions, pages, options []string
	form.
		Strings(prefix+"Source", &sources).
		Strings(prefix+"Expression", &expressions).
		Strings(prefix+"Pages", &pages).
		Strings(prefix+"Options", &options)

	at := func(values []string, i int) string {
		if i < len(values) {
			return values[i]
		}
		return ""
	}

	stamps := make([]gotenberg.Stamp, 0, len(sources))
	for i, source := range sources {
		if source != gotenberg.StampSourceText && source != gotenberg.StampSourceImage && source != gotenberg.StampSourcePDF {
			return nil, api.WrapError(
				fmt.Errorf("wrong %sSource value '%s'", prefix, source),
				api.NewSentinelHttpError(
					http.StatusBadRequest,
					fmt.Sprintf("Invalid form data: form field '%sSource' is invalid (got '%s', resulting to wrong value, expected either '%s', '%s' or '%s')", prefix, source, gotenberg.StampSourceText, gotenberg.StampSourceImage, gotenberg.StampSourcePDF),
				),
			)
		}

		var opts map[string]string
		if raw := at(options, i); raw != "" {
			err := json.Unmarshal([]byte(raw), &opts)
			if err != nil {
				return nil, api.WrapError(
					fmt.Errorf("unmarshal %sOptions: %w", prefix, err),
					api.NewSentinelHttpError(
						http.StatusBadRequest,
						fmt.Sprintf("Invalid form data: form field '%sOptions' is invalid", prefix),
					),
				)
			}
		}

		stamps = append(stamps, gotenberg.Stamp{
			Source:     source,
			Expression: at(expressions, i),
			Pages:      at(pages, i),
			Options:    opts,
		})
	}

	return stamps, nil
}

// BindStampFiles assigns each image or pdf stamp its uploaded stamp file. See
// [bindStampOrWatermarkFiles].
func BindStampFiles(stamps []gotenberg.Stamp, stampFiles []string) error {
	return bindStampOrWatermarkFiles(stamps, stampFiles, "stamp")
}

// BindWatermarkFiles assigns each image or pdf watermark its uploaded watermark
// file. See [bindStampOrWatermarkFiles].
func BindWatermarkFiles(watermarks []gotenberg.Stamp, watermarkFiles []string) error {
	return bindStampOrWatermarkFiles(watermarks, watermarkFiles, "watermark")
}

// bindStampOrWatermarkFiles assigns each image or pdf entry its uploaded file,
// consuming files in order. Text entries take no file. It returns an [api] HTTP
// 400 error when an image or pdf entry has no file left to consume, which also
// prevents an anonymous caller from passing an arbitrary filesystem path via
// the expression field. kind is "stamp" or "watermark" and shapes the error.
func bindStampOrWatermarkFiles(stamps []gotenberg.Stamp, files []string, kind string) error {
	fileIndex := 0
	for i := range stamps {
		if stamps[i].Source != gotenberg.StampSourceImage && stamps[i].Source != gotenberg.StampSourcePDF {
			continue
		}
		if fileIndex >= len(files) {
			return api.WrapError(
				fmt.Errorf("not enough %s files for the image or pdf entries", kind),
				api.NewSentinelHttpError(
					http.StatusBadRequest,
					fmt.Sprintf("Invalid form data: a %s file is required for image or pdf source", kind),
				),
			)
		}
		stamps[i].Expression = files[fileIndex]
		fileIndex++
	}

	return nil
}

// WatermarkStub applies each watermark to a list of PDF files, in order.
// Entries with no source are skipped, so an empty list does nothing.
func WatermarkStub(ctx *api.Context, engine gotenberg.PdfEngine, watermarks []gotenberg.Stamp, inputPaths []string) error {
	for _, watermark := range watermarks {
		if watermark.Source == "" {
			continue
		}

		for _, inputPath := range inputPaths {
			err := engine.Watermark(ctx, ctx.Log(), inputPath, watermark)
			if err != nil {
				return fmt.Errorf("watermark '%s': %w", inputPath, err)
			}
		}
	}

	return nil
}

// StampStub applies each stamp to a list of PDF files, in order. Entries with
// no source are skipped, so an empty list does nothing.
func StampStub(ctx *api.Context, engine gotenberg.PdfEngine, stamps []gotenberg.Stamp, inputPaths []string) error {
	for _, stamp := range stamps {
		if stamp.Source == "" {
			continue
		}

		for _, inputPath := range inputPaths {
			err := engine.Stamp(ctx, ctx.Log(), inputPath, stamp)
			if err != nil {
				return fmt.Errorf("stamp '%s': %w", inputPath, err)
			}
		}
	}

	return nil
}

// mergeRoute returns an [api.Route] which can merge PDFs.
func mergeRoute(engine gotenberg.PdfEngine) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/pdfengines/merge",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)

			form := ctx.FormData()
			pdfFormats := FormDataPdfFormats(form)
			metadata := FormDataPdfMetadata(form, false)
			bookmarks := FormDataPdfBookmarks(form, false)
			encrypt := FormDataPdfEncrypt(form)
			embedPaths := FormDataPdfEmbeds(form)
			watermarks, wErr := FormDataPdfWatermarks(form)
			if wErr != nil {
				return fmt.Errorf("form data watermarks: %w", wErr)
			}
			stamps, sErr := FormDataPdfStamps(form)
			if sErr != nil {
				return fmt.Errorf("form data stamps: %w", sErr)
			}
			var watermarkFiles, stampFiles []string
			form.Watermarks(&watermarkFiles).Stamps(&stampFiles)
			angle, rotatePages := FormDataPdfRotate(form, false)
			embedsMetadata := FormDataPdfEmbedsMetadata(form)
			facturX, facturxXmlPath := FormDataPdfFacturX(form)
			optimizeImages, imageQuality := FormDataPdfOptimize(form)

			var inputPaths []string
			var flatten bool
			var autoIndexBookmarks bool
			var titleBookmarks bool
			err := form.
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				Bool("flatten", &flatten, false).
				Bool("autoIndexBookmarks", &autoIndexBookmarks, false).
				Bool("titleBookmarks", &titleBookmarks, false).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			err = BindWatermarkFiles(watermarks, watermarkFiles)
			if err != nil {
				return fmt.Errorf("bind watermark files: %w", err)
			}
			err = BindStampFiles(stamps, stampFiles)
			if err != nil {
				return fmt.Errorf("bind stamp files: %w", err)
			}

			err = ValidatePdfFormatsCompat(pdfFormats, encrypt.UserPassword, embedPaths)
			if err != nil {
				return err
			}

			err = ValidatePdfEncryptCompat(encrypt)
			if err != nil {
				return err
			}

			err = ValidateFacturXCompat(facturX, facturxXmlPath, pdfFormats)
			if err != nil {
				return err
			}

			outputPath := ctx.GeneratePath(".pdf")
			err = engine.Merge(ctx, ctx.Log(), inputPaths, outputPath)
			if err != nil {
				return fmt.Errorf("merge PDFs: %w", err)
			}

			outputPaths := []string{outputPath}

			err = WatermarkStub(ctx, engine, watermarks, outputPaths)
			if err != nil {
				return fmt.Errorf("watermark PDFs: %w", err)
			}

			err = StampStub(ctx, engine, stamps, outputPaths)
			if err != nil {
				return fmt.Errorf("stamp PDFs: %w", err)
			}

			err = RotateStub(ctx, engine, angle, rotatePages, outputPaths)
			if err != nil {
				return fmt.Errorf("rotate PDFs: %w", err)
			}

			if flatten {
				err = FlattenStub(ctx, engine, outputPaths)
				if err != nil {
					return fmt.Errorf("flatten PDFs: %w", err)
				}
			}

			err = OptimizeStub(ctx, engine, optimizeImages, imageQuality, outputPaths)
			if err != nil {
				return fmt.Errorf("optimize PDF images: %w", err)
			}

			pdfFormats = FacturXPdfFormats(ctx, engine, facturX, pdfFormats, false, outputPaths)

			outputPaths, err = ConvertStub(ctx, engine, pdfFormats, outputPaths)
			if err != nil {
				return fmt.Errorf("convert PDF: %w", err)
			}

			// Bookmarks, metadata, and embeds are written after Convert,
			// as LibreOffice strips them during PDF/A conversion.

			var finalBookmarks []gotenberg.Bookmark
			if b, ok := bookmarks.([]gotenberg.Bookmark); ok {
				finalBookmarks = b
			} else {
				bMap, _ := bookmarks.(map[string][]gotenberg.Bookmark)
				if bMap != nil || autoIndexBookmarks || titleBookmarks {
					offset := 0
					for _, inputPath := range inputPaths {
						filename := ctx.OriginalFilename(inputPath)

						var fileBookmarks []gotenberg.Bookmark
						if bMap != nil {
							fileBookmarks = bMap[filename]
						}

						// titleBookmarks nests each input's own outline under
						// its title entry, so its outline is read here too.
						if len(fileBookmarks) == 0 && (autoIndexBookmarks || titleBookmarks) {
							fb, err := engine.ReadBookmarks(ctx, ctx.Log(), inputPath)
							if err != nil {
								return fmt.Errorf("read bookmarks of '%s': %w", filename, err)
							}
							fileBookmarks = fb
						}

						fileBookmarks = shiftBookmarks(fileBookmarks, offset)

						if titleBookmarks {
							finalBookmarks = append(finalBookmarks, gotenberg.Bookmark{
								Title:    documentTitle(ctx, engine, inputPath, filename),
								Page:     offset + 1,
								Children: fileBookmarks,
							})
						} else if len(fileBookmarks) > 0 {
							finalBookmarks = append(finalBookmarks, fileBookmarks...)
						}

						pageCount, err := engine.PageCount(ctx, ctx.Log(), inputPath)
						if err != nil {
							return fmt.Errorf("get page count of '%s': %w", filename, err)
						}
						offset += pageCount
					}
				}
			}

			if len(finalBookmarks) > 0 {
				err = WriteBookmarksStub(ctx, engine, finalBookmarks, outputPaths)
				if err != nil {
					return fmt.Errorf("write bookmarks: %w", err)
				}
			}

			err = WriteMetadataStub(ctx, engine, metadata, outputPaths)
			if err != nil {
				return fmt.Errorf("write metadata: %w", err)
			}

			err = EmbedFilesStub(ctx, engine, embedPaths, outputPaths)
			if err != nil {
				return fmt.Errorf("embed files into PDFs: %w", err)
			}

			err = EmbedFilesMetadataStub(ctx, engine, embedsMetadata, outputPaths)
			if err != nil {
				return fmt.Errorf("set embeds metadata: %w", err)
			}

			err = ApplyFacturXStub(ctx, engine, facturX, facturxXmlPath, outputPaths)
			if err != nil {
				return fmt.Errorf("apply Factur-X: %w", err)
			}

			err = EncryptPdfStub(ctx, engine, encrypt, outputPaths)
			if err != nil {
				return fmt.Errorf("encrypt PDFs: %w", err)
			}

			err = ctx.AddOutputPaths(outputPaths...)
			if err != nil {
				return fmt.Errorf("add output paths: %w", err)
			}

			return nil
		},
	}
}

// splitRoute returns an [api.Route] which can extract pages from a PDF.
func splitRoute(engine gotenberg.PdfEngine) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/pdfengines/split",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)

			form := ctx.FormData()
			mode := FormDataPdfSplitMode(form, true)
			pdfFormats := FormDataPdfFormats(form)
			metadata := FormDataPdfMetadata(form, false)
			encrypt := FormDataPdfEncrypt(form)
			embedPaths := FormDataPdfEmbeds(form)
			watermarks, wErr := FormDataPdfWatermarks(form)
			if wErr != nil {
				return fmt.Errorf("form data watermarks: %w", wErr)
			}
			stamps, sErr := FormDataPdfStamps(form)
			if sErr != nil {
				return fmt.Errorf("form data stamps: %w", sErr)
			}
			var watermarkFiles, stampFiles []string
			form.Watermarks(&watermarkFiles).Stamps(&stampFiles)
			angle, rotatePages := FormDataPdfRotate(form, false)
			embedsMetadata := FormDataPdfEmbedsMetadata(form)
			facturX, facturxXmlPath := FormDataPdfFacturX(form)
			optimizeImages, imageQuality := FormDataPdfOptimize(form)

			var inputPaths []string
			var flatten bool
			err := form.
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				Bool("flatten", &flatten, false).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			err = BindWatermarkFiles(watermarks, watermarkFiles)
			if err != nil {
				return fmt.Errorf("bind watermark files: %w", err)
			}
			err = BindStampFiles(stamps, stampFiles)
			if err != nil {
				return fmt.Errorf("bind stamp files: %w", err)
			}

			err = ValidatePdfFormatsCompat(pdfFormats, encrypt.UserPassword, embedPaths)
			if err != nil {
				return err
			}

			err = ValidatePdfEncryptCompat(encrypt)
			if err != nil {
				return err
			}

			err = ValidateFacturXCompat(facturX, facturxXmlPath, pdfFormats)
			if err != nil {
				return err
			}

			outputPaths, err := SplitPdfStub(ctx, engine, mode, inputPaths)
			if err != nil {
				return fmt.Errorf("split PDFs: %w", err)
			}

			err = WatermarkStub(ctx, engine, watermarks, outputPaths)
			if err != nil {
				return fmt.Errorf("watermark PDFs: %w", err)
			}

			err = StampStub(ctx, engine, stamps, outputPaths)
			if err != nil {
				return fmt.Errorf("stamp PDFs: %w", err)
			}

			err = RotateStub(ctx, engine, angle, rotatePages, outputPaths)
			if err != nil {
				return fmt.Errorf("rotate PDFs: %w", err)
			}

			if flatten {
				err = FlattenStub(ctx, engine, outputPaths)
				if err != nil {
					return fmt.Errorf("flatten PDFs: %w", err)
				}
			}

			err = OptimizeStub(ctx, engine, optimizeImages, imageQuality, outputPaths)
			if err != nil {
				return fmt.Errorf("optimize PDF images: %w", err)
			}

			pdfFormats = FacturXPdfFormats(ctx, engine, facturX, pdfFormats, false, outputPaths)

			convertOutputPaths, err := ConvertStub(ctx, engine, pdfFormats, outputPaths)
			if err != nil {
				return fmt.Errorf("convert PDFs: %w", err)
			}

			// Metadata, embeds are written after Convert, as LibreOffice
			// strips them during PDF/A conversion.
			err = WriteMetadataStub(ctx, engine, metadata, convertOutputPaths)
			if err != nil {
				return fmt.Errorf("write metadata: %w", err)
			}

			err = EmbedFilesStub(ctx, engine, embedPaths, convertOutputPaths)
			if err != nil {
				return fmt.Errorf("embed files into PDFs: %w", err)
			}

			err = EmbedFilesMetadataStub(ctx, engine, embedsMetadata, convertOutputPaths)
			if err != nil {
				return fmt.Errorf("set embeds metadata: %w", err)
			}

			err = ApplyFacturXStub(ctx, engine, facturX, facturxXmlPath, convertOutputPaths)
			if err != nil {
				return fmt.Errorf("apply Factur-X: %w", err)
			}

			err = EncryptPdfStub(ctx, engine, encrypt, convertOutputPaths)
			if err != nil {
				return fmt.Errorf("encrypt PDFs: %w", err)
			}

			zeroValuedSplitMode := gotenberg.SplitMode{}
			zeroValuedPdfFormats := gotenberg.PdfFormats{}
			if mode != zeroValuedSplitMode && pdfFormats != zeroValuedPdfFormats {
				// Rename the files to keep the split naming.
				for i, convertOutputPath := range convertOutputPaths {
					err = ctx.Rename(convertOutputPath, outputPaths[i])
					if err != nil {
						return fmt.Errorf("rename output path: %w", err)
					}
				}
			}

			err = ctx.AddOutputPaths(outputPaths...)
			if err != nil {
				return fmt.Errorf("add output paths: %w", err)
			}

			return nil
		},
	}
}

// flattenRoute returns an [api.Route] which can flatten PDFs.
func flattenRoute(engine gotenberg.PdfEngine) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/pdfengines/flatten",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)

			form := ctx.FormData()

			var inputPaths []string
			err := form.
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			err = FlattenStub(ctx, engine, inputPaths)
			if err != nil {
				return fmt.Errorf("flatten PDFs: %w", err)
			}

			err = ctx.AddOutputPaths(inputPaths...)
			if err != nil {
				return fmt.Errorf("add output paths: %w", err)
			}

			return nil
		},
	}
}

// convertRoute returns an [api.Route] which can convert PDFs to a specific ODF
// format.
func optimizeRoute(engine gotenberg.PdfEngine) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/pdfengines/optimize",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)

			form := ctx.FormData()
			// This route optimizes unconditionally, so the optimizeImages toggle
			// is ignored; only the image quality is read.
			_, imageQuality := FormDataPdfOptimize(form)

			var inputPaths []string
			err := form.
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			err = OptimizeStub(ctx, engine, true, imageQuality, inputPaths)
			if err != nil {
				return fmt.Errorf("optimize PDF images: %w", err)
			}

			err = ctx.AddOutputPaths(inputPaths...)
			if err != nil {
				return fmt.Errorf("add output paths: %w", err)
			}

			return nil
		},
	}
}

func convertRoute(engine gotenberg.PdfEngine) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/pdfengines/convert",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)

			form := ctx.FormData()
			pdfFormats := FormDataPdfFormats(form)

			var inputPaths []string
			err := form.
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			zeroValued := gotenberg.PdfFormats{}
			if pdfFormats == zeroValued {
				return api.WrapError(
					errors.New("no PDF formats"),
					api.NewSentinelHttpError(
						http.StatusBadRequest,
						"Invalid form data: either 'pdfa' or 'pdfua' form fields must be provided",
					),
				)
			}

			outputPaths, err := ConvertStub(ctx, engine, pdfFormats, inputPaths)
			if err != nil {
				return fmt.Errorf("convert PDFs: %w", err)
			}

			if len(outputPaths) > 1 {
				// If .zip archive, keep the original filename.
				for i, inputPath := range inputPaths {
					err = ctx.Rename(outputPaths[i], inputPath)
					if err != nil {
						return fmt.Errorf("rename output path: %w", err)
					}
					outputPaths[i] = inputPath
				}
			}

			err = ctx.AddOutputPaths(outputPaths...)
			if err != nil {
				return fmt.Errorf("add output paths: %w", err)
			}

			return nil
		},
	}
}

// readMetadataRoute returns an [api.Route] which returns the metadata of PDFs.
func readMetadataRoute(engine gotenberg.PdfEngine) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/pdfengines/metadata/read",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)

			var inputPaths []string
			err := ctx.FormData().
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			res := make(map[string]map[string]any, len(inputPaths))
			for _, inputPath := range inputPaths {
				metadata, err := engine.ReadMetadata(ctx, ctx.Log(), inputPath)
				if err != nil {
					return fmt.Errorf("read metadata: %w", err)
				}

				res[ctx.OriginalFilename(inputPath)] = metadata
			}

			err = c.JSON(http.StatusOK, res)
			if err != nil {
				if strings.Contains(err.Error(), "request method or response status code does not allow body") {
					// High probability that the user is using the webhook
					// feature. It does not make sense for this route.
					return api.ErrNoOutputFile
				}
				return fmt.Errorf("return JSON response: %w", err)
			}

			return api.ErrNoOutputFile
		},
	}
}

// writeMetadataRoute returns an [api.Route] which can write metadata into
// PDFs.
func writeMetadataRoute(engine gotenberg.PdfEngine) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/pdfengines/metadata/write",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)

			form := ctx.FormData()
			metadata := FormDataPdfMetadata(form, true)

			var inputPaths []string
			err := form.
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			err = WriteMetadataStub(ctx, engine, metadata, inputPaths)
			if err != nil {
				return fmt.Errorf("write metadata: %w", err)
			}

			err = ctx.AddOutputPaths(inputPaths...)
			if err != nil {
				return fmt.Errorf("add output paths: %w", err)
			}

			return nil
		},
	}
}

// readBookmarksRoute returns an [api.Route] which returns the bookmarks of PDFs.
func readBookmarksRoute(engine gotenberg.PdfEngine) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/pdfengines/bookmarks/read",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)

			var inputPaths []string
			err := ctx.FormData().
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			res := make(map[string][]gotenberg.Bookmark, len(inputPaths))
			for _, inputPath := range inputPaths {
				bookmarks, err := engine.ReadBookmarks(ctx, ctx.Log(), inputPath)
				if err != nil {
					return fmt.Errorf("read bookmarks: %w", err)
				}

				res[ctx.OriginalFilename(inputPath)] = bookmarks
			}

			err = c.JSON(http.StatusOK, res)
			if err != nil {
				if strings.Contains(err.Error(), "request method or response status code does not allow body") {
					// High probability that the user is using the webhook
					// feature. It does not make sense for this route.
					return api.ErrNoOutputFile
				}
				return fmt.Errorf("return JSON response: %w", err)
			}

			return api.ErrNoOutputFile
		},
	}
}

// writeBookmarksRoute returns an [api.Route] which can write bookmarks into PDFs.
func writeBookmarksRoute(engine gotenberg.PdfEngine) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/pdfengines/bookmarks/write",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)

			form := ctx.FormData()
			bookmarks := FormDataPdfBookmarks(form, true)

			var inputPaths []string
			err := form.
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			err = WriteBookmarksStub(ctx, engine, bookmarks, inputPaths)
			if err != nil {
				return fmt.Errorf("write bookmarks: %w", err)
			}

			err = ctx.AddOutputPaths(inputPaths...)
			if err != nil {
				return fmt.Errorf("add output paths: %w", err)
			}

			return nil
		},
	}
}

// encryptRoute returns an [api.Route] which can add password protection to PDFs.
func encryptRoute(engine gotenberg.PdfEngine) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/pdfengines/encrypt",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)

			form := ctx.FormData()
			encrypt := FormDataPdfEncrypt(form)

			var inputPaths []string
			err := form.
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			// At least one password is required; an empty user password with an
			// owner password yields an owner-only document.
			if encrypt.UserPassword == "" && encrypt.OwnerPassword == "" {
				return api.WrapError(
					errors.New("no password provided"),
					api.NewSentinelHttpError(http.StatusBadRequest, "Invalid form data: a 'userPassword' or 'ownerPassword' is required"),
				)
			}

			err = EncryptPdfStub(ctx, engine, encrypt, inputPaths)
			if err != nil {
				return fmt.Errorf("encrypt PDFs: %w", err)
			}

			err = ctx.AddOutputPaths(inputPaths...)
			if err != nil {
				return fmt.Errorf("add output paths: %w", err)
			}

			return nil
		},
	}
}

// embedRoute returns an [api.Route] which can add embedded files to PDFs.
// TODO: attachments instead?
func embedRoute(engine gotenberg.PdfEngine) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/pdfengines/embed",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)

			form := ctx.FormData()
			embedPaths := FormDataPdfEmbeds(form)
			embedsMetadata := FormDataPdfEmbedsMetadata(form)
			facturX, facturxXmlPath := FormDataPdfFacturX(form)

			var inputPaths []string
			err := form.
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			err = ValidateFacturXCompat(facturX, facturxXmlPath, gotenberg.PdfFormats{})
			if err != nil {
				return err
			}

			// Factur-X requires PDF/A-3. Convert when needed; a no-op otherwise,
			// so a plain embed request keeps its inputs untouched.
			pdfFormats := FacturXPdfFormats(ctx, engine, facturX, gotenberg.PdfFormats{}, false, inputPaths)
			outputPaths, err := ConvertStub(ctx, engine, pdfFormats, inputPaths)
			if err != nil {
				return fmt.Errorf("convert PDFs: %w", err)
			}

			err = EmbedFilesStub(ctx, engine, embedPaths, outputPaths)
			if err != nil {
				return fmt.Errorf("embed files into PDFs: %w", err)
			}

			err = EmbedFilesMetadataStub(ctx, engine, embedsMetadata, outputPaths)
			if err != nil {
				return fmt.Errorf("set embeds metadata: %w", err)
			}

			err = ApplyFacturXStub(ctx, engine, facturX, facturxXmlPath, outputPaths)
			if err != nil {
				return fmt.Errorf("apply Factur-X: %w", err)
			}

			err = ctx.AddOutputPaths(outputPaths...)
			if err != nil {
				return fmt.Errorf("add output paths: %w", err)
			}

			return nil
		},
	}
}

// watermarkRoute returns an [api.Route] which can add watermarks to PDFs.
//
//nolint:dupl
func watermarkRoute(engine gotenberg.PdfEngine) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/pdfengines/watermark",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)

			form := ctx.FormData()
			watermarks, err := FormDataPdfWatermarks(form)
			if err != nil {
				return fmt.Errorf("form data watermarks: %w", err)
			}

			var inputPaths []string
			var watermarkFiles []string
			err = form.
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				Watermarks(&watermarkFiles).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			if len(watermarks) == 0 {
				return api.WrapError(
					errors.New("no watermark provided"),
					api.NewSentinelHttpError(
						http.StatusBadRequest,
						"Invalid form data: form field 'watermarkSource' is required",
					),
				)
			}

			err = BindWatermarkFiles(watermarks, watermarkFiles)
			if err != nil {
				return fmt.Errorf("bind watermark files: %w", err)
			}

			err = WatermarkStub(ctx, engine, watermarks, inputPaths)
			if err != nil {
				return fmt.Errorf("watermark PDFs: %w", err)
			}

			err = ctx.AddOutputPaths(inputPaths...)
			if err != nil {
				return fmt.Errorf("add output paths: %w", err)
			}

			return nil
		},
	}
}

// stampRoute returns an [api.Route] which can add stamps to PDFs.
//
//nolint:dupl
func stampRoute(engine gotenberg.PdfEngine) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/pdfengines/stamp",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)

			form := ctx.FormData()

			// Reading the stamp fields as parallel arrays applies several
			// stamps in one request. A single occurrence of each field is the
			// existing single-stamp behavior.
			stamps, err := FormDataPdfStamps(form)
			if err != nil {
				return fmt.Errorf("form data stamps: %w", err)
			}

			var inputPaths []string
			var stampFiles []string
			err = form.
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				Stamps(&stampFiles).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			if len(stamps) == 0 {
				return api.WrapError(
					errors.New("no stamp provided"),
					api.NewSentinelHttpError(
						http.StatusBadRequest,
						"Invalid form data: form field 'stampSource' is required",
					),
				)
			}

			err = BindStampFiles(stamps, stampFiles)
			if err != nil {
				return fmt.Errorf("bind stamp files: %w", err)
			}

			err = StampStub(ctx, engine, stamps, inputPaths)
			if err != nil {
				return fmt.Errorf("stamp PDFs: %w", err)
			}

			err = ctx.AddOutputPaths(inputPaths...)
			if err != nil {
				return fmt.Errorf("add output paths: %w", err)
			}

			return nil
		},
	}
}

// rotateRoute returns an [api.Route] which can rotate pages of PDFs.
func rotateRoute(engine gotenberg.PdfEngine) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/pdfengines/rotate",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)

			form := ctx.FormData()
			angle, pages := FormDataPdfRotate(form, true)

			var inputPaths []string
			err := form.
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			err = RotateStub(ctx, engine, angle, pages, inputPaths)
			if err != nil {
				return fmt.Errorf("rotate PDFs: %w", err)
			}

			err = ctx.AddOutputPaths(inputPaths...)
			if err != nil {
				return fmt.Errorf("add output paths: %w", err)
			}

			return nil
		},
	}
}

// facturXRoute returns an [api.Route] which turns existing PDFs into Factur-X
// documents: it ensures PDF/A-3, embeds the CII invoice XML, and injects the fx
// XMP metadata.
func facturXRoute(engine gotenberg.PdfEngine) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/pdfengines/factur-x",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)

			form := ctx.FormData()
			pdfFormats := FormDataPdfFormats(form)
			facturX, facturxXmlPath := FormDataPdfFacturX(form)

			var inputPaths []string
			err := form.
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			// Factur-X is the whole point of this route, so both fields are
			// mandatory here.
			if facturX.ConformanceLevel == "" || facturxXmlPath == "" {
				return api.WrapError(
					errors.New("facturxConformanceLevel and facturxXml are required"),
					api.NewSentinelHttpError(http.StatusBadRequest, "Invalid form data: 'facturxConformanceLevel' and 'facturxXml' are both required"),
				)
			}

			err = ValidateFacturXCompat(facturX, facturxXmlPath, pdfFormats)
			if err != nil {
				return err
			}

			pdfFormats = FacturXPdfFormats(ctx, engine, facturX, pdfFormats, false, inputPaths)
			outputPaths, err := ConvertStub(ctx, engine, pdfFormats, inputPaths)
			if err != nil {
				return fmt.Errorf("convert PDFs: %w", err)
			}

			err = ApplyFacturXStub(ctx, engine, facturX, facturxXmlPath, outputPaths)
			if err != nil {
				return fmt.Errorf("apply Factur-X: %w", err)
			}

			err = ctx.AddOutputPaths(outputPaths...)
			if err != nil {
				return fmt.Errorf("add output paths: %w", err)
			}

			return nil
		},
	}
}
