package libreoffice

import (
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"

	"github.com/labstack/echo/v5"

	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
	libreofficeapi "github.com/gotenberg/gotenberg/v9/pkg/modules/libreoffice/api"
	"github.com/gotenberg/gotenberg/v9/pkg/modules/pdf"
)

// unattributableFailureMessage is returned when LibreOffice fails and no
// client-supplied input is implicated. Its only format verb is the original
// filename.
const unattributableFailureMessage = "LibreOffice failed to convert the document '%s'. This is usually a resource issue: increase the container's memory and CPU, or reduce the document's size. The request is valid and may be retried."

// convertRoute returns an [api.Route] which can convert LibreOffice documents
// to PDF.
func convertRoute(libreOffice libreofficeapi.Converter, processor *pdf.Processor) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/libreoffice/convert",
		IsMultipart: true,
		Handler: func(c *echo.Context) error {
			ctx := c.Get("context").(*api.Context)
			form := ctx.FormData()
			stages, files, err := pdf.FormDataStages(form)
			if err != nil {
				return err
			}

			var inputPaths []string
			options := libreofficeapi.DefaultOptions()

			// A paper size sets the width and height, which paperWidth and
			// paperHeight then override one at a time. Without any, each
			// document keeps its own page size.
			form.PaperSize("paperSize", &options.PaperWidth, &options.PaperHeight)

			err = form.
				MandatoryPaths(libreOffice.Extensions(), &inputPaths).
				String("password", &options.Password).
				Bool("landscape", &options.Landscape).
				Inches("paperWidth", &options.PaperWidth).
				Inches("paperHeight", &options.PaperHeight).
				String("nativePageRanges", &options.PageRanges).
				Bool("updateIndexes", &options.UpdateIndexes).
				Bool("exportFormFields", &options.ExportFormFields).
				Bool("allowDuplicateFieldNames", &options.AllowDuplicateFieldNames).
				Bool("nativeBookmarks", &options.ExportBookmarks).
				Bool("exportBookmarksToPdfDestination", &options.ExportBookmarksToPdfDestination).
				Bool("exportPlaceholders", &options.ExportPlaceholders).
				Bool("exportNotes", &options.ExportNotes).
				Bool("exportNotesPages", &options.ExportNotesPages).
				Bool("exportOnlyNotesPages", &options.ExportOnlyNotesPages).
				Bool("exportNotesInMargin", &options.ExportNotesInMargin).
				Bool("convertOooTargetToPdfTarget", &options.ConvertOooTargetToPdfTarget).
				Bool("exportLinksRelativeFsys", &options.ExportLinksRelativeFsys).
				Bool("exportHiddenSlides", &options.ExportHiddenSlides).
				Bool("skipEmptyPages", &options.SkipEmptyPages).
				Bool("addOriginalDocumentAsStream", &options.AddOriginalDocumentAsStream).
				Bool("singlePage", &options.SinglePageSheets).
				Bool("nativeTaggedPdf", &options.GenerateTaggedPdf).
				Int("initialView", &options.InitialView, api.OneOf(0, 1, 2)).
				Int("initialPage", &options.InitialPage, api.Min(1)).
				Int("magnification", &options.Magnification, api.OneOf(0, 1, 2, 3, 4)).
				Int("zoom", &options.Zoom, api.Min(1)).
				Int("pageLayout", &options.PageLayout, api.OneOf(0, 1, 2, 3)).
				Bool("firstPageOnLeft", &options.FirstPageOnLeft).
				Bool("resizeWindowToInitialPage", &options.ResizeWindowToInitialPage).
				Bool("centerWindow", &options.CenterWindow).
				Bool("openInFullScreenMode", &options.OpenInFullScreenMode).
				Bool("displayPDFDocumentTitle", &options.DisplayPDFDocumentTitle).
				Bool("hideViewerMenubar", &options.HideViewerMenubar).
				Bool("hideViewerToolbar", &options.HideViewerToolbar).
				Bool("hideViewerWindowControls", &options.HideViewerWindowControls).
				Bool("useTransitionEffects", &options.UseTransitionEffects).
				Int("openBookmarkLevels", &options.OpenBookmarkLevels, func(value int) error {
					if value != -1 && (value < 1 || value > 10) {
						return errors.New("value is not -1 or between 1 and 10")
					}
					return nil
				}).
				Bool("nativeLosslessImageCompression", &options.LosslessImageCompression).
				Int("nativeQuality", &options.Quality, api.Min(1), api.Max(100)).
				Bool("nativeReduceImageResolution", &options.ReduceImageResolution).
				Int("nativeMaxImageResolution", &options.MaxImageResolution, api.OneOf(75, 150, 300, 600, 1200)).
				String("nativeWatermarkText", &options.NativeWatermarkText).
				Int("nativeWatermarkColor", &options.NativeWatermarkColor).
				Int("nativeWatermarkFontHeight", &options.NativeWatermarkFontHeight, api.Min(0)).
				Int("nativeWatermarkRotateAngle", &options.NativeWatermarkRotateAngle).
				String("nativeWatermarkFontName", &options.NativeWatermarkFontName).
				String("nativeTiledWatermarkText", &options.NativeTiledWatermarkText).
				String("nativePdfa", &options.PdfFormats.PdfA).
				Bool("nativePdfua", &options.PdfFormats.PdfUa).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			if (options.PaperWidth == 0) != (options.PaperHeight == 0) {
				return api.NewSentinelHttpError(
					http.StatusBadRequest,
					"Invalid form data: set 'paperSize', or both 'paperWidth' and 'paperHeight'",
				)
			}

			// A file a stage reads, such as a stamp image, is not a document to
			// convert, although LibreOffice may read its format.
			referenced := slices.Collect(maps.Values(files))
			inputPaths = slices.DeleteFunc(inputPaths, func(path string) bool {
				return slices.Contains(referenced, path)
			})
			if len(inputPaths) == 0 {
				return api.NewSentinelHttpError(
					http.StatusBadRequest,
					"Invalid form data: no document to convert; every uploaded file is read by a stage",
				)
			}

			err = options.PdfFormats.Validate()
			if err != nil {
				return api.WrapError(
					fmt.Errorf("validate PDF formats: %w", err),
					api.NewSentinelHttpError(
						http.StatusBadRequest,
						fmt.Sprintf("Invalid form data: form field 'nativePdfa' is invalid (got '%s', want PDF/A-1b, PDF/A-2b, PDF/A-3b or PDF/A-4).", options.PdfFormats.PdfA),
					),
				)
			}

			outputPaths := make([]string, len(inputPaths))
			for i, inputPath := range inputPaths {
				// document.docx -> document.docx.pdf, which names the output,
				// or the archive entry, and the parts of a split.
				outputPaths[i] = ctx.GeneratePathFromFilename(ctx.OriginalFilename(inputPath) + ".pdf")

				err = libreOffice.Pdf(ctx, ctx.Log(), inputPath, outputPaths[i], options)
				if err != nil {
					filename := ctx.OriginalFilename(inputPath)

					// Blame the client only when one of its inputs is actually
					// implicated, since the server is the only remaining
					// explanation otherwise. A password failure aborts on
					// import, before the export filter applies any page range.
					// See https://github.com/gotenberg/gotenberg/issues/1588.
					var sentinel api.SentinelHttpError
					switch {
					case errors.Is(err, libreofficeapi.ErrInvalidPassword):
						sentinel = api.NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("The password for the document '%s' is incorrect. Check the 'password' form field.", filename))
					case errors.Is(err, libreofficeapi.ErrLoadFailed) && options.Password == "" && libreofficeapi.DetectPasswordProtection(inputPath) == libreofficeapi.PasswordProtectionRequired:
						sentinel = api.NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("The document '%s' is password-protected. Provide its password in the 'password' form field.", filename))
					case errors.Is(err, libreofficeapi.ErrLoadFailed):
						sentinel = api.NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("LibreOffice could not read the document '%s'. Ensure the file is not corrupted and that its extension matches its actual format.", filename))
					case errors.Is(err, libreofficeapi.ErrSaveFailed) && options.PageRanges != "":
						sentinel = api.NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("LibreOffice could not apply the page ranges '%s' to the document '%s'. Check the 'nativePageRanges' form field; valid values look like '1-4', '2' or '1,3,5-7'.", options.PageRanges, filename))
					case errors.Is(err, libreofficeapi.ErrSaveFailed), errors.Is(err, libreofficeapi.ErrCrashed):
						sentinel = api.NewSentinelHttpError(http.StatusInternalServerError, fmt.Sprintf(unattributableFailureMessage, filename))
					default:
						return fmt.Errorf("convert to PDF: %w", err)
					}

					return api.WrapError(fmt.Errorf("convert to PDF: %w", err), sentinel)
				}
			}

			outputPaths, err = processor.Run(ctx, stages, outputPaths, files)
			if err != nil {
				return fmt.Errorf("run stages: %w", err)
			}

			err = ctx.AddOutputPaths(outputPaths...)
			if err != nil {
				return fmt.Errorf("add output paths: %w", err)
			}

			return nil
		},
	}
}
