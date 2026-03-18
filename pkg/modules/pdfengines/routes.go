package pdfengines

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

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
		inputPathNoExt := inputPath[:len(inputPath)-len(filepath.Ext(inputPath))]
		filenameNoExt := filepath.Base(inputPathNoExt)
		outputDirPath, err := ctx.CreateSubDirectory(strings.ReplaceAll(filepath.Base(filenameNoExt), ".", "_"))
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
			if mode.Unify && mode.Mode == gotenberg.SplitModePages {
				newPath = fmt.Sprintf(
					"%s/%s.pdf",
					outputDirPath, filenameNoExt,
				)
			} else {
				newPath = fmt.Sprintf(
					"%s/%s_%d.pdf",
					outputDirPath, filenameNoExt, i,
				)
			}

			err = ctx.Rename(path, newPath)
			if err != nil {
				return nil, fmt.Errorf("rename path: %w", err)
			}

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
			filename := filepath.Base(inputPath)
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

// FormDataPdfEncrypt extracts encryption parameters from form data.
func FormDataPdfEncrypt(form *api.FormData) (userPassword, ownerPassword string) {
	form.String("userPassword", &userPassword, "")
	form.String("ownerPassword", &ownerPassword, "")
	return userPassword, ownerPassword
}

// EncryptPdfStub adds password protection to PDF files.
func EncryptPdfStub(ctx *api.Context, engine gotenberg.PdfEngine, userPassword, ownerPassword string, inputPaths []string) error {
	if userPassword == "" {
		return nil
	}

	for _, inputPath := range inputPaths {
		err := engine.Encrypt(ctx, ctx.Log(), inputPath, userPassword, ownerPassword)
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

	for _, inputPath := range inputPaths {
		err := engine.EmbedFiles(ctx, ctx.Log(), embedPaths, inputPath)
		if err != nil {
			return fmt.Errorf("embed files into PDF '%s': %w", inputPath, err)
		}
	}

	return nil
}

// FormDataPdfWatermark creates a [gotenberg.Stamp] for watermarking from the
// form data.
func FormDataPdfWatermark(form *api.FormData, mandatory bool) gotenberg.Stamp {
	return formDataPdfStampOrWatermark(form, "watermark", mandatory)
}

// FormDataPdfStamp creates a [gotenberg.Stamp] for stamping from the form data.
func FormDataPdfStamp(form *api.FormData, mandatory bool) gotenberg.Stamp {
	return formDataPdfStampOrWatermark(form, "stamp", mandatory)
}

func formDataPdfStampOrWatermark(form *api.FormData, prefix string, mandatory bool) gotenberg.Stamp {
	var (
		source     string
		expression string
		pages      string
		options    map[string]string
	)

	sourceFunc := func(value string) error {
		if value != "" && value != gotenberg.StampSourceText && value != gotenberg.StampSourceImage && value != gotenberg.StampSourcePDF {
			return fmt.Errorf("wrong value, expected either '%s', '%s' or '%s'", gotenberg.StampSourceText, gotenberg.StampSourceImage, gotenberg.StampSourcePDF)
		}
		source = value
		return nil
	}

	optionsFunc := func(value string) error {
		if value == "" {
			return nil
		}
		err := json.Unmarshal([]byte(value), &options)
		if err != nil {
			return fmt.Errorf("unmarshal %s options: %w", prefix, err)
		}
		return nil
	}

	if mandatory {
		form.
			MandatoryCustom(prefix+"Source", func(value string) error {
				return sourceFunc(value)
			}).
			String(prefix+"Expression", &expression, "").
			String(prefix+"Pages", &pages, "").
			Custom(prefix+"Options", func(value string) error {
				return optionsFunc(value)
			})
	} else {
		form.
			Custom(prefix+"Source", func(value string) error {
				return sourceFunc(value)
			}).
			String(prefix+"Expression", &expression, "").
			String(prefix+"Pages", &pages, "").
			Custom(prefix+"Options", func(value string) error {
				return optionsFunc(value)
			})
	}

	return gotenberg.Stamp{
		Source:     source,
		Expression: expression,
		Pages:      pages,
		Options:    options,
	}
}

// FormDataPdfWatermarkFiles extracts watermark file paths from form data.
func FormDataPdfWatermarkFiles(form *api.FormData) []string {
	var paths []string
	form.Watermarks(&paths)
	return paths
}

// FormDataPdfStampFiles extracts stamp file paths from form data.
func FormDataPdfStampFiles(form *api.FormData) []string {
	var paths []string
	form.Stamps(&paths)
	return paths
}

// WatermarkStub applies a watermark to a list of PDF files. If the stamp has
// no source, it does nothing.
func WatermarkStub(ctx *api.Context, engine gotenberg.PdfEngine, stamp gotenberg.Stamp, inputPaths []string) error {
	if stamp.Source == "" {
		return nil
	}

	for _, inputPath := range inputPaths {
		err := engine.Watermark(ctx, ctx.Log(), inputPath, stamp)
		if err != nil {
			return fmt.Errorf("watermark '%s': %w", inputPath, err)
		}
	}

	return nil
}

// StampStub applies a stamp to a list of PDF files. If the stamp has
// no source, it does nothing.
func StampStub(ctx *api.Context, engine gotenberg.PdfEngine, stamp gotenberg.Stamp, inputPaths []string) error {
	if stamp.Source == "" {
		return nil
	}

	for _, inputPath := range inputPaths {
		err := engine.Stamp(ctx, ctx.Log(), inputPath, stamp)
		if err != nil {
			return fmt.Errorf("stamp '%s': %w", inputPath, err)
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
			userPassword, ownerPassword := FormDataPdfEncrypt(form)
			embedPaths := FormDataPdfEmbeds(form)
			watermark := FormDataPdfWatermark(form, false)
			watermarkFiles := FormDataPdfWatermarkFiles(form)
			stamp := FormDataPdfStamp(form, false)
			stampFiles := FormDataPdfStampFiles(form)

			var inputPaths []string
			var flatten bool
			var autoIndexBookmarks bool
			err := form.
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				Bool("flatten", &flatten, false).
				Bool("autoIndexBookmarks", &autoIndexBookmarks, false).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			if (watermark.Source == gotenberg.StampSourceImage || watermark.Source == gotenberg.StampSourcePDF) && len(watermarkFiles) > 0 {
				watermark.Expression = watermarkFiles[0]
			}
			if (stamp.Source == gotenberg.StampSourceImage || stamp.Source == gotenberg.StampSourcePDF) && len(stampFiles) > 0 {
				stamp.Expression = stampFiles[0]
			}

			err = ValidatePdfFormatsCompat(pdfFormats, userPassword, embedPaths)
			if err != nil {
				return err
			}

			outputPath := ctx.GeneratePath(".pdf")
			err = engine.Merge(ctx, ctx.Log(), inputPaths, outputPath)
			if err != nil {
				return fmt.Errorf("merge PDFs: %w", err)
			}

			outputPaths := []string{outputPath}

			err = WatermarkStub(ctx, engine, watermark, outputPaths)
			if err != nil {
				return fmt.Errorf("watermark PDFs: %w", err)
			}

			err = StampStub(ctx, engine, stamp, outputPaths)
			if err != nil {
				return fmt.Errorf("stamp PDFs: %w", err)
			}

			if flatten {
				err = FlattenStub(ctx, engine, outputPaths)
				if err != nil {
					return fmt.Errorf("flatten PDFs: %w", err)
				}
			}

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
				if bMap != nil || autoIndexBookmarks {
					offset := 0
					for _, inputPath := range inputPaths {
						filename := filepath.Base(inputPath)

						var fileBookmarks []gotenberg.Bookmark
						if bMap != nil {
							fileBookmarks = bMap[filename]
						}

						if len(fileBookmarks) == 0 && autoIndexBookmarks {
							fb, err := engine.ReadBookmarks(ctx, ctx.Log(), inputPath)
							if err != nil {
								return fmt.Errorf("read bookmarks of '%s': %w", filename, err)
							}
							fileBookmarks = fb
						}

						if len(fileBookmarks) > 0 {
							finalBookmarks = append(finalBookmarks, shiftBookmarks(fileBookmarks, offset)...)
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

			err = EncryptPdfStub(ctx, engine, userPassword, ownerPassword, outputPaths)
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
			userPassword, ownerPassword := FormDataPdfEncrypt(form)
			embedPaths := FormDataPdfEmbeds(form)
			watermark := FormDataPdfWatermark(form, false)
			watermarkFiles := FormDataPdfWatermarkFiles(form)
			stamp := FormDataPdfStamp(form, false)
			stampFiles := FormDataPdfStampFiles(form)

			var inputPaths []string
			var flatten bool
			err := form.
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				Bool("flatten", &flatten, false).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			if (watermark.Source == gotenberg.StampSourceImage || watermark.Source == gotenberg.StampSourcePDF) && len(watermarkFiles) > 0 {
				watermark.Expression = watermarkFiles[0]
			}
			if (stamp.Source == gotenberg.StampSourceImage || stamp.Source == gotenberg.StampSourcePDF) && len(stampFiles) > 0 {
				stamp.Expression = stampFiles[0]
			}

			err = ValidatePdfFormatsCompat(pdfFormats, userPassword, embedPaths)
			if err != nil {
				return err
			}

			outputPaths, err := SplitPdfStub(ctx, engine, mode, inputPaths)
			if err != nil {
				return fmt.Errorf("split PDFs: %w", err)
			}

			err = WatermarkStub(ctx, engine, watermark, outputPaths)
			if err != nil {
				return fmt.Errorf("watermark PDFs: %w", err)
			}

			err = StampStub(ctx, engine, stamp, outputPaths)
			if err != nil {
				return fmt.Errorf("stamp PDFs: %w", err)
			}

			if flatten {
				err = FlattenStub(ctx, engine, outputPaths)
				if err != nil {
					return fmt.Errorf("flatten PDFs: %w", err)
				}
			}

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

			err = EncryptPdfStub(ctx, engine, userPassword, ownerPassword, convertOutputPaths)
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

				res[filepath.Base(inputPath)] = metadata
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

				res[filepath.Base(inputPath)] = bookmarks
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

			var inputPaths []string
			var userPassword string
			var ownerPassword string
			err := form.
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				MandatoryString("userPassword", &userPassword).
				String("ownerPassword", &ownerPassword, "").
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			err = EncryptPdfStub(ctx, engine, userPassword, ownerPassword, inputPaths)
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

			var inputPaths []string
			err := form.
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}
			err = EmbedFilesStub(ctx, engine, embedPaths, inputPaths)
			if err != nil {
				return fmt.Errorf("embed files into PDFs: %w", err)
			}

			err = ctx.AddOutputPaths(inputPaths...)
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
			stamp := FormDataPdfWatermark(form, true)
			watermarkFiles := FormDataPdfWatermarkFiles(form)

			var inputPaths []string
			err := form.
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			if stamp.Source == gotenberg.StampSourceImage || stamp.Source == gotenberg.StampSourcePDF {
				if len(watermarkFiles) == 0 {
					return api.WrapError(
						errors.New("no watermark file provided"),
						api.NewSentinelHttpError(
							http.StatusBadRequest,
							"Invalid form data: a watermark file is required for image or pdf source",
						),
					)
				}
				stamp.Expression = watermarkFiles[0]
			}

			err = WatermarkStub(ctx, engine, stamp, inputPaths)
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
			stamp := FormDataPdfStamp(form, true)
			stampFiles := FormDataPdfStampFiles(form)

			var inputPaths []string
			err := form.
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			if stamp.Source == gotenberg.StampSourceImage || stamp.Source == gotenberg.StampSourcePDF {
				if len(stampFiles) == 0 {
					return api.WrapError(
						errors.New("no stamp file provided"),
						api.NewSentinelHttpError(
							http.StatusBadRequest,
							"Invalid form data: a stamp file is required for image or pdf source",
						),
					)
				}
				stamp.Expression = stampFiles[0]
			}

			err = StampStub(ctx, engine, stamp, inputPaths)
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
