package libreoffice

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/api"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/libreoffice/unoconv"
)

// convertRoute returns an api.MultipartFormDataRoute which can convert
// LibreOffice documents to PDF.
func convertRoute(uno unoconv.API, engine gotenberg.PDFEngine) api.MultipartFormDataRoute {
	return api.MultipartFormDataRoute{
		Path: "/libreoffice/convert",
		Handler: func(ctx *api.Context) error {
			// Let's get the data from the form and validate them.
			var (
				inputPaths         []string
				landscape          bool
				nativePageRanges   string
				nativePDFA1aFormat bool
				PDFformat          string
				merge              bool
			)

			err := ctx.FormData().
				MandatoryPaths(uno.Extensions(), &inputPaths).
				Bool("landscape", &landscape, false).
				String("nativePageRanges", &nativePageRanges, "").
				Bool("nativePdfA1aFormat", &nativePDFA1aFormat, false).
				String("pdfFormat", &PDFformat, "").
				Bool("merge", &merge, false).
				Validate()

			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			if nativePDFA1aFormat && PDFformat != "" {
				return api.WrapError(
					errors.New("got both 'pdfFormat' and 'nativePdfA1aFormat' form values"),
					api.NewSentinelHTTPError(http.StatusBadRequest, "Both 'pdfFormat' and 'nativePdfA1aFormat' form values are provided"),
				)
			}

			// Alright, let's convert each document to PDF.

			outputPaths := make([]string, len(inputPaths))

			for i, inputPath := range inputPaths {
				outputPaths[i] = ctx.GeneratePath(".pdf")

				options := unoconv.Options{
					Landscape:  landscape,
					PageRanges: nativePageRanges,
					PDFArchive: nativePDFA1aFormat,
				}

				err = uno.PDF(ctx, ctx.Log(), inputPath, outputPaths[i], options)

				if err != nil {
					if errors.Is(err, unoconv.ErrMalformedPageRanges) {
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

				// Note: nativePdfA1aFormat has not been specified if we reach
				// this part of the code. Indeed, the handler returns early on
				// an error if both nativePdfA1aFormat and pdfFormat are
				// present.

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
			}

			// Ok, we don't have to merge the PDFs. Let's check if the client
			// want to convert each PDF to a specific PDF format.

			// Note: nativePdfA1aFormat has not been specified if we reach this
			// part of the code. Indeed, the handler returns early on an error
			// if both nativePdfA1aFormat and pdfFormat are present.

			if PDFformat != "" {
				convertOutputPaths := make([]string, len(outputPaths))

				for i, outputPath := range outputPaths {
					convertInputPath := outputPath
					convertOutputPaths[i] = ctx.GeneratePath(".pdf")

					err = engine.Convert(ctx, ctx.Log(), PDFformat, convertInputPath, convertOutputPaths[i])

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

				// Important: the output paths are now the converted files.
				outputPaths = convertOutputPaths
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
