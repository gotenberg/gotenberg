package libreoffice

import (
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strconv"

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
func convertRoute(libreOffice libreofficeapi.Uno, processor *pdf.Processor) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/libreoffice/convert",
		IsMultipart: true,
		Handler: func(c *echo.Context) error {
			ctx := c.Get("context").(*api.Context)
			defaultOptions := libreofficeapi.DefaultOptions()

			form := ctx.FormData()
			stages, files, err := pdf.FormDataStages(form)
			if err != nil {
				return err
			}

			var (
				inputPaths                      []string
				password                        string
				landscape                       bool
				nativePageRanges                string
				updateIndexes                   bool
				exportFormFields                bool
				allowDuplicateFieldNames        bool
				exportBookmarks                 bool
				exportBookmarksToPdfDestination bool
				exportPlaceholders              bool
				exportNotes                     bool
				exportNotesPages                bool
				exportOnlyNotesPages            bool
				exportNotesInMargin             bool
				convertOooTargetToPdfTarget     bool
				exportLinksRelativeFsys         bool
				exportHiddenSlides              bool
				skipEmptyPages                  bool
				addOriginalDocumentAsStream     bool
				singlePageSheets                bool
				generateTaggedPdf               bool
				losslessImageCompression        bool
				quality                         int
				reduceImageResolution           bool
				maxImageResolution              int
				nativeWatermarkText             string
				nativeWatermarkColor            int
				initialView                     int
				initialPage                     int
				magnification                   int
				zoom                            int
				pageLayout                      int
				firstPageOnLeft                 bool
				resizeWindowToInitialPage       bool
				centerWindow                    bool
				openInFullScreenMode            bool
				displayPDFDocumentTitle         bool
				hideViewerMenubar               bool
				hideViewerToolbar               bool
				hideViewerWindowControls        bool
				useTransitionEffects            bool
				openBookmarkLevels              int
				nativeWatermarkFontHeight       int
				nativeWatermarkRotateAngle      int
				nativeWatermarkFontName         string
				nativeTiledWatermarkText        string
				pdfa                            string
				pdfua                           bool
			)

			err = form.
				MandatoryPaths(libreOffice.Extensions(), &inputPaths).
				String("password", &password, defaultOptions.Password).
				Bool("landscape", &landscape, defaultOptions.Landscape).
				String("nativePageRanges", &nativePageRanges, defaultOptions.PageRanges).
				Bool("updateIndexes", &updateIndexes, defaultOptions.UpdateIndexes).
				Bool("exportFormFields", &exportFormFields, defaultOptions.ExportFormFields).
				Bool("allowDuplicateFieldNames", &allowDuplicateFieldNames, defaultOptions.AllowDuplicateFieldNames).
				Bool("exportBookmarks", &exportBookmarks, defaultOptions.ExportBookmarks).
				Bool("exportBookmarksToPdfDestination", &exportBookmarksToPdfDestination, defaultOptions.ExportBookmarksToPdfDestination).
				Bool("exportPlaceholders", &exportPlaceholders, defaultOptions.ExportPlaceholders).
				Bool("exportNotes", &exportNotes, defaultOptions.ExportNotes).
				Bool("exportNotesPages", &exportNotesPages, defaultOptions.ExportNotesPages).
				Bool("exportOnlyNotesPages", &exportOnlyNotesPages, defaultOptions.ExportOnlyNotesPages).
				Bool("exportNotesInMargin", &exportNotesInMargin, defaultOptions.ExportNotesInMargin).
				Bool("convertOooTargetToPdfTarget", &convertOooTargetToPdfTarget, defaultOptions.ConvertOooTargetToPdfTarget).
				Bool("exportLinksRelativeFsys", &exportLinksRelativeFsys, defaultOptions.ExportLinksRelativeFsys).
				Bool("exportHiddenSlides", &exportHiddenSlides, defaultOptions.ExportHiddenSlides).
				Bool("skipEmptyPages", &skipEmptyPages, defaultOptions.SkipEmptyPages).
				Bool("addOriginalDocumentAsStream", &addOriginalDocumentAsStream, defaultOptions.AddOriginalDocumentAsStream).
				Bool("singlePageSheets", &singlePageSheets, defaultOptions.SinglePageSheets).
				Bool("generateTaggedPdf", &generateTaggedPdf, defaultOptions.GenerateTaggedPdf).
				Custom("initialView", func(value string) error {
					if value == "" {
						initialView = defaultOptions.InitialView
						return nil
					}
					intValue, err := strconv.Atoi(value)
					if err != nil {
						return err
					}
					if !slices.Contains([]int{0, 1, 2}, intValue) {
						return errors.New("value is not 0, 1 or 2")
					}
					initialView = intValue
					return nil
				}).
				Custom("initialPage", func(value string) error {
					if value == "" {
						initialPage = defaultOptions.InitialPage
						return nil
					}
					intValue, err := strconv.Atoi(value)
					if err != nil {
						return err
					}
					if intValue < 1 {
						return errors.New("value is inferior to 1")
					}
					initialPage = intValue
					return nil
				}).
				Custom("magnification", func(value string) error {
					if value == "" {
						magnification = defaultOptions.Magnification
						return nil
					}
					intValue, err := strconv.Atoi(value)
					if err != nil {
						return err
					}
					if !slices.Contains([]int{0, 1, 2, 3, 4}, intValue) {
						return errors.New("value is not 0, 1, 2, 3 or 4")
					}
					magnification = intValue
					return nil
				}).
				Custom("zoom", func(value string) error {
					if value == "" {
						zoom = defaultOptions.Zoom
						return nil
					}
					intValue, err := strconv.Atoi(value)
					if err != nil {
						return err
					}
					if intValue < 1 {
						return errors.New("value is inferior to 1")
					}
					zoom = intValue
					return nil
				}).
				Custom("pageLayout", func(value string) error {
					if value == "" {
						pageLayout = defaultOptions.PageLayout
						return nil
					}
					intValue, err := strconv.Atoi(value)
					if err != nil {
						return err
					}
					if !slices.Contains([]int{0, 1, 2, 3}, intValue) {
						return errors.New("value is not 0, 1, 2 or 3")
					}
					pageLayout = intValue
					return nil
				}).
				Bool("firstPageOnLeft", &firstPageOnLeft, defaultOptions.FirstPageOnLeft).
				Bool("resizeWindowToInitialPage", &resizeWindowToInitialPage, defaultOptions.ResizeWindowToInitialPage).
				Bool("centerWindow", &centerWindow, defaultOptions.CenterWindow).
				Bool("openInFullScreenMode", &openInFullScreenMode, defaultOptions.OpenInFullScreenMode).
				Bool("displayPDFDocumentTitle", &displayPDFDocumentTitle, defaultOptions.DisplayPDFDocumentTitle).
				Bool("hideViewerMenubar", &hideViewerMenubar, defaultOptions.HideViewerMenubar).
				Bool("hideViewerToolbar", &hideViewerToolbar, defaultOptions.HideViewerToolbar).
				Bool("hideViewerWindowControls", &hideViewerWindowControls, defaultOptions.HideViewerWindowControls).
				Bool("useTransitionEffects", &useTransitionEffects, defaultOptions.UseTransitionEffects).
				Custom("openBookmarkLevels", func(value string) error {
					if value == "" {
						openBookmarkLevels = defaultOptions.OpenBookmarkLevels
						return nil
					}
					intValue, err := strconv.Atoi(value)
					if err != nil {
						return err
					}
					if intValue != -1 && (intValue < 1 || intValue > 10) {
						return errors.New("value is not -1 or between 1 and 10")
					}
					openBookmarkLevels = intValue
					return nil
				}).
				Bool("losslessImageCompression", &losslessImageCompression, defaultOptions.LosslessImageCompression).
				Custom("quality", func(value string) error {
					if value == "" {
						quality = defaultOptions.Quality
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

					quality = intValue
					return nil
				}).
				Bool("reduceImageResolution", &reduceImageResolution, defaultOptions.ReduceImageResolution).
				Custom("maxImageResolution", func(value string) error {
					if value == "" {
						maxImageResolution = defaultOptions.MaxImageResolution
						return nil
					}

					intValue, err := strconv.Atoi(value)
					if err != nil {
						return err
					}

					if !slices.Contains([]int{75, 150, 300, 600, 1200}, intValue) {
						return errors.New("value is not 75, 150, 300, 600 or 1200")
					}

					maxImageResolution = intValue
					return nil
				}).
				String("nativeWatermarkText", &nativeWatermarkText, defaultOptions.NativeWatermarkText).
				Custom("nativeWatermarkColor", func(value string) error {
					if value == "" {
						nativeWatermarkColor = defaultOptions.NativeWatermarkColor
						return nil
					}
					intValue, err := strconv.Atoi(value)
					if err != nil {
						return err
					}
					nativeWatermarkColor = intValue
					return nil
				}).
				Custom("nativeWatermarkFontHeight", func(value string) error {
					if value == "" {
						nativeWatermarkFontHeight = defaultOptions.NativeWatermarkFontHeight
						return nil
					}
					intValue, err := strconv.Atoi(value)
					if err != nil {
						return err
					}
					if intValue < 0 {
						return errors.New("value is inferior to 0")
					}
					nativeWatermarkFontHeight = intValue
					return nil
				}).
				Custom("nativeWatermarkRotateAngle", func(value string) error {
					if value == "" {
						nativeWatermarkRotateAngle = defaultOptions.NativeWatermarkRotateAngle
						return nil
					}
					intValue, err := strconv.Atoi(value)
					if err != nil {
						return err
					}
					nativeWatermarkRotateAngle = intValue
					return nil
				}).
				String("nativeWatermarkFontName", &nativeWatermarkFontName, defaultOptions.NativeWatermarkFontName).
				String("nativeTiledWatermarkText", &nativeTiledWatermarkText, defaultOptions.NativeTiledWatermarkText).
				String("pdfa", &pdfa, "").
				Bool("pdfua", &pdfua, false).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
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

			pdfFormats := libreofficeapi.PdfFormats{PdfA: pdfa, PdfUa: pdfua}

			outputPaths := make([]string, len(inputPaths))
			for i, inputPath := range inputPaths {
				// document.docx -> document.docx.pdf, which names the output,
				// or the archive entry, and the parts of a split.
				outputPaths[i] = ctx.GeneratePathFromFilename(ctx.OriginalFilename(inputPath) + ".pdf")
				options := libreofficeapi.Options{
					Password:                        password,
					Landscape:                       landscape,
					PageRanges:                      nativePageRanges,
					UpdateIndexes:                   updateIndexes,
					ExportFormFields:                exportFormFields,
					AllowDuplicateFieldNames:        allowDuplicateFieldNames,
					ExportBookmarks:                 exportBookmarks,
					ExportBookmarksToPdfDestination: exportBookmarksToPdfDestination,
					ExportPlaceholders:              exportPlaceholders,
					ExportNotes:                     exportNotes,
					ExportNotesPages:                exportNotesPages,
					ExportOnlyNotesPages:            exportOnlyNotesPages,
					ExportNotesInMargin:             exportNotesInMargin,
					ConvertOooTargetToPdfTarget:     convertOooTargetToPdfTarget,
					ExportLinksRelativeFsys:         exportLinksRelativeFsys,
					ExportHiddenSlides:              exportHiddenSlides,
					SkipEmptyPages:                  skipEmptyPages,
					AddOriginalDocumentAsStream:     addOriginalDocumentAsStream,
					SinglePageSheets:                singlePageSheets,
					InitialView:                     initialView,
					InitialPage:                     initialPage,
					Magnification:                   magnification,
					Zoom:                            zoom,
					PageLayout:                      pageLayout,
					FirstPageOnLeft:                 firstPageOnLeft,
					ResizeWindowToInitialPage:       resizeWindowToInitialPage,
					CenterWindow:                    centerWindow,
					OpenInFullScreenMode:            openInFullScreenMode,
					DisplayPDFDocumentTitle:         displayPDFDocumentTitle,
					HideViewerMenubar:               hideViewerMenubar,
					HideViewerToolbar:               hideViewerToolbar,
					HideViewerWindowControls:        hideViewerWindowControls,
					UseTransitionEffects:            useTransitionEffects,
					OpenBookmarkLevels:              openBookmarkLevels,
					LosslessImageCompression:        losslessImageCompression,
					Quality:                         quality,
					ReduceImageResolution:           reduceImageResolution,
					MaxImageResolution:              maxImageResolution,
					NativeWatermarkText:             nativeWatermarkText,
					NativeWatermarkColor:            nativeWatermarkColor,
					NativeWatermarkFontHeight:       nativeWatermarkFontHeight,
					NativeWatermarkRotateAngle:      nativeWatermarkRotateAngle,
					NativeWatermarkFontName:         nativeWatermarkFontName,
					NativeTiledWatermarkText:        nativeTiledWatermarkText,
					GenerateTaggedPdf:               generateTaggedPdf,
					PdfFormats:                      pdfFormats,
				}

				err = libreOffice.Pdf(ctx, ctx.Log(), inputPath, outputPaths[i], options)
				if err != nil {
					if errors.Is(err, libreofficeapi.ErrInvalidPdfFormats) {
						return api.WrapError(
							fmt.Errorf("convert to PDF: %w", err),
							api.NewSentinelHttpError(
								http.StatusBadRequest,
								fmt.Sprintf("The PDF format '%s' is not supported. Valid formats include PDF/A-1b, PDF/A-2b, PDF/A-3b, PDF/A-4, and PDF/UA.", pdfFormats.PdfA),
							),
						)
					}

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
