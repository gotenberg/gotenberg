package pdfengines

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/labstack/echo/v4"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

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
func FormDataPdfMetadata(form *api.FormData) map[string]interface{} {
	var metadata map[string]interface{}
	form.Custom("metadata", func(value string) error {
		if len(value) > 0 {
			err := json.Unmarshal([]byte(value), &metadata)
			if err != nil {
				return fmt.Errorf("unmarshal metadata: %w", err)
			}
		}
		return nil
	})
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
			metadata := FormDataPdfMetadata(form)

			var inputPaths []string
			err := form.
				MandatoryPaths([]string{".pdf"}, &inputPaths).
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

			err = ctx.AddOutputPaths(outputPaths...)
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

			var (
				inputPaths []string
				metadata   map[string]interface{}
			)

			err := ctx.FormData().
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				MandatoryCustom("metadata", func(value string) error {
					if len(value) > 0 {
						err := json.Unmarshal([]byte(value), &metadata)
						if err != nil {
							return fmt.Errorf("unmarshal metadata: %w", err)
						}
					}
					if len(metadata) == 0 {
						return errors.New("no metadata")
					}
					return nil
				}).
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
