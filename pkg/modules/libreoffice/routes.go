package libreoffice

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/api"
	libreofficeapi "github.com/gotenberg/gotenberg/v7/pkg/modules/libreoffice/api"
)

// convertRoute returns an [api.Route] which can convert LibreOffice documents
// to PDF.
func convertRoute(libreOffice libreofficeapi.Uno, engine gotenberg.PDFEngine) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/libreoffice/convert",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)

			// Let's get the data from the form and validate them.
			var (
				inputPaths         []string
				landscape          bool
				nativePageRanges   string
				nativePdfA1aFormat bool
				nativePdfFormat    string
				pdfFormat          string
				merge              bool
			)

			err := ctx.FormData().
				MandatoryPaths(libreOffice.Extensions(), &inputPaths).
				Bool("landscape", &landscape, false).
				String("nativePageRanges", &nativePageRanges, "").
				Bool("nativePdfA1aFormat", &nativePdfA1aFormat, false).
				String("nativePdfFormat", &nativePdfFormat, "").
				String("pdfFormat", &pdfFormat, "").
				Bool("merge", &merge, false).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			// nativePdfFormat > pdfFormat > nativePdfA1aFormat.
			var (
				actualPdfFormat string
				nativeFormat    bool
			)

			// FIXME: deprecated.
			if nativePdfA1aFormat {
				ctx.Log().Warn("'nativePdfA1aFormat' is deprecated; prefer 'nativePdfFormat' or 'pdfFormat' form fields instead")
				actualPdfFormat = gotenberg.FormatPDFA1a
				nativeFormat = true
			}

			if pdfFormat != "" {
				actualPdfFormat = pdfFormat
				nativeFormat = false
			}

			if nativePdfFormat != "" {
				actualPdfFormat = nativePdfFormat
				nativeFormat = true
			}

			// Alright, let's convert each document to PDF.

			outputPaths := make([]string, len(inputPaths))

			for i, inputPath := range inputPaths {
				outputPaths[i] = ctx.GeneratePath(".pdf")

				options := libreofficeapi.Options{
					Landscape:  landscape,
					PageRanges: nativePageRanges,
					PdfFormat:  nativePdfFormat,
				}

				err = libreOffice.Pdf(ctx, ctx.Log(), inputPath, outputPaths[i], options)

				if err != nil {
					if errors.Is(err, libreofficeapi.ErrMalformedPageRanges) {
						return api.WrapError(
							fmt.Errorf("convert to PDF: %w", err),
							api.NewSentinelHTTPError(http.StatusBadRequest, fmt.Sprintf("Malformed page ranges '%s' (nativePageRanges)", options.PageRanges)),
						)
					}

					return fmt.Errorf("convert to PDF: %w", err)
				}
			}

			// So far so good, let's check if we have to merge the PDFs. Quick
			// win: if there is only one PDF, skip this step.

			if len(outputPaths) > 1 && merge {
				outputPath := ctx.GeneratePath(".pdf")

				err = engine.Merge(ctx, ctx.Log(), outputPaths, outputPath)
				if err != nil {
					return fmt.Errorf("merge PDFs: %w", err)
				}

				// Now, let's check if the client want to convert this result
				// PDF to a specific PDF format.

				if !nativeFormat && actualPdfFormat != "" {
					convertInputPath := outputPath
					convertOutputPath := ctx.GeneratePath(".pdf")

					err = engine.Convert(ctx, ctx.Log(), actualPdfFormat, convertInputPath, convertOutputPath)

					if err != nil {
						if errors.Is(err, gotenberg.ErrPDFFormatNotAvailable) {
							return api.WrapError(
								fmt.Errorf("convert PDF: %w", err),
								api.NewSentinelHTTPError(
									http.StatusBadRequest,
									fmt.Sprintf("At least one PDF engine does not handle the PDF format '%s' (pdfFormat), while other have failed to convert for other reasons", actualPdfFormat),
								),
							)
						}

						return fmt.Errorf("convert PDF: %w", err)
					}

					// Important: the output path is now the converted file.
					outputPath = convertOutputPath
				}

				// Last but not least, add the output path to the context so that
				// the Uno is able to send it as a response to the client.

				err = ctx.AddOutputPaths(outputPath)
				if err != nil {
					return fmt.Errorf("add output path: %w", err)
				}

				return nil
			}

			// Ok, we don't have to merge the PDFs. Let's check if the client
			// want to convert each PDF to a specific PDF format.

			if !nativeFormat && actualPdfFormat != "" {
				convertOutputPaths := make([]string, len(outputPaths))

				for i, outputPath := range outputPaths {
					convertInputPath := outputPath
					convertOutputPaths[i] = ctx.GeneratePath(".pdf")

					err = engine.Convert(ctx, ctx.Log(), actualPdfFormat, convertInputPath, convertOutputPaths[i])

					if err != nil {
						if errors.Is(err, gotenberg.ErrPDFFormatNotAvailable) {
							return api.WrapError(
								fmt.Errorf("convert PDF: %w", err),
								api.NewSentinelHTTPError(
									http.StatusBadRequest,
									fmt.Sprintf("At least one PDF engine does not handle the PDF format '%s' (pdfFormat), while other have failed to convert for other reasons", actualPdfFormat),
								),
							)
						}

						return fmt.Errorf("convert PDF: %w", err)
					}

				}

				// Important: the output paths are now the converted files.
				outputPaths = convertOutputPaths
			}

			// Last but not least, add the output paths to the context so that
			// the Uno is able to send them as a response to the client.

			err = ctx.AddOutputPaths(outputPaths...)
			if err != nil {
				return fmt.Errorf("add output paths: %w", err)
			}

			return nil
		},
	}
}
