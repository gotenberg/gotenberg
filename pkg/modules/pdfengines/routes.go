package pdfengines

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/api"
	"github.com/labstack/echo/v4"
)

// mergeRoute returns an api.Route which can merge PDFs.
func mergeRoute(engine gotenberg.PDFEngine) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/pdfengines/merge",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)

			// Let's get the data from the form and validate them.
			var (
				inputPaths []string
				PDFformat  string
			)

			err := ctx.FormData().
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				String("pdfFormat", &PDFformat, "").
				Validate()

			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			// Alright, let's merge the PDFs.

			outputPath := ctx.GeneratePath(".pdf")

			err = engine.Merge(ctx, ctx.Log(), inputPaths, outputPath)
			if err != nil {
				return fmt.Errorf("merge PDFs: %w", err)
			}

			// So far so good, the PDFs are merged into one unique PDF.
			// Now, let's check if the client want to convert this result PDF
			// to a specific PDF format.

			if PDFformat != "" {
				convertInputPath := outputPath
				convertOutputPath := ctx.GeneratePath(".pdf")

				err = engine.Convert(ctx, ctx.Log(), PDFformat, convertInputPath, convertOutputPath)

				if err != nil {
					if errors.Is(err, gotenberg.ErrPDFFormatNotAvailable) {
						return api.WrapError(
							fmt.Errorf("convert PDF: %w", err),
							api.NewSentinelHTTPError(
								http.StatusBadRequest,
								fmt.Sprintf("At least one PDF engine does not handle the PDF format '%s' (pdfFormat), while other have failed to convert for other reasons", PDFformat),
							),
						)
					}

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

// convertRoute returns an api.Route which can convert a PDF to a specific PDF
// format.
func convertRoute(engine gotenberg.PDFEngine) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/pdfengines/convert",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)

			// Let's get the data from the form and validate them.
			var (
				inputPaths []string
				PDFformat  string
			)

			err := ctx.FormData().
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				MandatoryString("pdfFormat", &PDFformat).
				Validate()

			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			// Alright, let's merge the PDFs.

			outputPaths := make([]string, len(inputPaths))

			for i, inputPath := range inputPaths {
				outputPaths[i] = ctx.GeneratePath(".pdf")

				err = engine.Convert(ctx, ctx.Log(), PDFformat, inputPath, outputPaths[i])

				if err != nil {
					if errors.Is(err, gotenberg.ErrPDFFormatNotAvailable) {
						return api.WrapError(
							fmt.Errorf("convert PDF: %w", err),
							api.NewSentinelHTTPError(
								http.StatusBadRequest,
								fmt.Sprintf("At least one PDF engine does not handle the PDF format '%s' (pdfFormat), while other have failed to convert for other reasons", PDFformat),
							),
						)
					}

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
