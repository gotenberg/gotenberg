package pdfengines

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

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

			// Alright, let's merge the PDFs.

			outputPath := ctx.GeneratePath("", ".pdf")

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
				convertOutputPath := ctx.GeneratePath("", ".pdf")

				err = engine.Convert(ctx, ctx.Log(), pdfFormats, convertInputPath, convertOutputPath)
				if err != nil {
					return fmt.Errorf("convert PDF: %w", err)
				}

				// Important: the output path is now the converted file.
				outputPath = convertOutputPath
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

// convertRoute returns an [api.Route] which can convert a PDF to a specific
// PDF format.
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

			// Alright, let's convert the PDFs.s
			outputPaths := make([]string, len(inputPaths))

			for i, inputPath := range inputPaths {
				if len(outputPaths) > 1 {
					// If .zip archive, keep the original filenames.
					outputPaths[i] = ctx.GeneratePath(strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath)), ".pdf")
				} else {
					outputPaths[i] = ctx.GeneratePath("", ".pdf")
				}

				err = engine.Convert(ctx, ctx.Log(), pdfFormats, inputPath, outputPaths[i])
				if err != nil {
					return fmt.Errorf("convert PDF: %w", err)
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
