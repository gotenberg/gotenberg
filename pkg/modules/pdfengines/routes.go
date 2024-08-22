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

// mergeRoute returns an [api.Route] which can merge PDFs.
func mergeRoute(engine gotenberg.PdfEngine) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/pdfengines/merge",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)

			// Let's get the data from the form and validate them.
			var (
				inputPaths []string
				pdfa       string
				pdfua      bool
				metadata   map[string]interface{}
			)

			err := ctx.FormData().
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				String("pdfa", &pdfa, "").
				Bool("pdfua", &pdfua, false).
				Custom("metadata", func(value string) error {
					if len(value) > 0 {
						err := json.Unmarshal([]byte(value), &metadata)
						if err != nil {
							return fmt.Errorf("unmarshal metadata: %w", err)
						}
					}
					return nil
				}).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			pdfFormats := gotenberg.PdfFormats{
				PdfA:  pdfa,
				PdfUa: pdfua,
			}

			// Alright, let's merge the PDFs.
			outputPath := ctx.GeneratePath(".pdf")

			err = engine.Merge(ctx, ctx.Log(), inputPaths, outputPath)
			if err != nil {
				return fmt.Errorf("merge PDFs: %w", err)
			}

			// So far so good, the PDFs are merged into one unique PDF.
			// Now, let's check if the client want to convert this result PDF
			// to specific PDF formats.
			zeroValued := gotenberg.PdfFormats{}
			if pdfFormats != zeroValued {
				convertInputPath := outputPath
				convertOutputPath := ctx.GeneratePath(".pdf")

				err = engine.Convert(ctx, ctx.Log(), pdfFormats, convertInputPath, convertOutputPath)
				if err != nil {
					return fmt.Errorf("convert PDF: %w", err)
				}

				// Important: the output path is now the converted file.
				outputPath = convertOutputPath
			}

			// Writes and potentially overrides metadata entries, if any.
			if len(metadata) > 0 {
				err = engine.WriteMetadata(ctx, ctx.Log(), metadata, outputPath)
				if err != nil {
					return fmt.Errorf("write metadata: %w", err)
				}
			}

			// Last but not least, add the output path to the context so that
			// the API is able to send it as a response to the client.
			err = ctx.AddOutputPaths(outputPath)
			if err != nil {
				return fmt.Errorf("add output path: %w", err)
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

			// Let's get the data from the form and validate them.
			var (
				inputPaths []string
				pdfa       string
				pdfua      bool
			)

			err := ctx.FormData().
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				String("pdfa", &pdfa, "").
				Bool("pdfua", &pdfua, false).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			pdfFormats := gotenberg.PdfFormats{
				PdfA:  pdfa,
				PdfUa: pdfua,
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

			// Alright, let's convert the PDFs.
			outputPaths := make([]string, len(inputPaths))
			for i, inputPath := range inputPaths {
				outputPaths[i] = ctx.GeneratePath(".pdf")

				err = engine.Convert(ctx, ctx.Log(), pdfFormats, inputPath, outputPaths[i])
				if err != nil {
					return fmt.Errorf("convert PDF: %w", err)
				}

				if len(outputPaths) > 1 {
					// If .zip archive, keep the original filename.
					err = ctx.Rename(outputPaths[i], inputPath)
					if err != nil {
						return fmt.Errorf("rename output path: %w", err)
					}

					outputPaths[i] = inputPath
				}
			}

			// Last but not least, add the output paths to the context so that
			// the API is able to send them as a response to the client.
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

			// Let's get the data from the form and validate them.
			var inputPaths []string

			err := ctx.FormData().
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			// Alright, let's read the metadata.
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

			// Let's get the data from the form and validate them.
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

			// Alright, let's convert the PDFs.
			for _, inputPath := range inputPaths {
				err = engine.WriteMetadata(ctx, ctx.Log(), metadata, inputPath)
				if err != nil {
					return fmt.Errorf("write metadata: %w", err)
				}
			}

			// Last but not least, add the output paths to the context so that
			// the API is able to send them as a response to the client.
			err = ctx.AddOutputPaths(inputPaths...)
			if err != nil {
				return fmt.Errorf("add output paths: %w", err)
			}

			return nil
		},
	}
}
