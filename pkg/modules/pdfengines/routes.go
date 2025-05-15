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
// Fallback to default value if the considered key is not present.
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
func FormDataPdfMetadata(form *api.FormData, mandatory bool) map[string]interface{} {
	var metadata map[string]interface{}

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
func WriteMetadataStub(ctx *api.Context, engine gotenberg.PdfEngine, metadata map[string]interface{}, inputPaths []string) error {
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

			var inputPaths []string
			var flatten bool
			err := form.
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				Bool("flatten", &flatten, false).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			outputPath := ctx.GeneratePath(".pdf")
			err = engine.Merge(ctx, ctx.Log(), inputPaths, outputPath)
			if err != nil {
				return fmt.Errorf("merge PDFs: %w", err)
			}

			outputPaths, err := ConvertStub(ctx, engine, pdfFormats, []string{outputPath})
			if err != nil {
				return fmt.Errorf("convert PDF: %w", err)
			}

			err = WriteMetadataStub(ctx, engine, metadata, outputPaths)
			if err != nil {
				return fmt.Errorf("write metadata: %w", err)
			}

			if flatten {
				err = FlattenStub(ctx, engine, outputPaths)
				if err != nil {
					return fmt.Errorf("flatten PDFs: %w", err)
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

			var inputPaths []string
			var flatten bool
			err := form.
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				Bool("flatten", &flatten, false).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			outputPaths, err := SplitPdfStub(ctx, engine, mode, inputPaths)
			if err != nil {
				return fmt.Errorf("split PDFs: %w", err)
			}

			convertOutputPaths, err := ConvertStub(ctx, engine, pdfFormats, outputPaths)
			if err != nil {
				return fmt.Errorf("convert PDFs: %w", err)
			}

			err = WriteMetadataStub(ctx, engine, metadata, convertOutputPaths)
			if err != nil {
				return fmt.Errorf("write metadata: %w", err)
			}

			if flatten {
				err = FlattenStub(ctx, engine, convertOutputPaths)
				if err != nil {
					return fmt.Errorf("flatten PDFs: %w", err)
				}
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

			res := make(map[string]map[string]interface{}, len(inputPaths))
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
