package libreoffice

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
	libreofficeapi "github.com/gotenberg/gotenberg/v8/pkg/modules/libreoffice/api"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/pdfengines"
)

// unattributableFailureMessage is returned when LibreOffice fails and no
// client-supplied input is implicated. Its only format verb is the original
// filename.
const unattributableFailureMessage = "LibreOffice failed to convert the document '%s'. This is usually a resource issue: increase the container's memory and CPU, or reduce the document's size. The request is valid and may be retried."

// convertRoute returns an [api.Route] which can convert LibreOffice documents
// to PDF.
func convertRoute(libreOffice libreofficeapi.Uno, engine gotenberg.PdfEngine) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/libreoffice/convert",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)
			defaultOptions := libreofficeapi.DefaultOptions()

			form := ctx.FormData()
			splitMode := pdfengines.FormDataPdfSplitMode(form, false)
			pdfFormats := pdfengines.FormDataPdfFormats(form)
			metadata := pdfengines.FormDataPdfMetadata(form, false)
			encrypt := pdfengines.FormDataPdfEncrypt(form)
			embedPaths := pdfengines.FormDataPdfEmbeds(form)
			watermark := pdfengines.FormDataPdfWatermark(form, false)
			watermarkFile := pdfengines.FormDataPdfWatermarkFile(form)
			stamp := pdfengines.FormDataPdfStamp(form, false)
			stampFile := pdfengines.FormDataPdfStampFile(form)
			angle, rotatePages := pdfengines.FormDataPdfRotate(form, false)
			embedsMetadata := pdfengines.FormDataPdfEmbedsMetadata(form)
			facturX, facturxXmlPath := pdfengines.FormDataPdfFacturX(form)
			optimizeImages, imageQuality := pdfengines.FormDataPdfOptimize(form)

			zeroValuedSplitMode := gotenberg.SplitMode{}

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
				nativePdfFormats                bool
				merge                           bool
				flatten                         bool
			)

			err := form.
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
				Bool("nativePdfFormats", &nativePdfFormats, true).
				Bool("merge", &merge, false).
				Bool("flatten", &flatten, false).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			err = pdfengines.EnsureWatermarkFile(&watermark, watermarkFile)
			if err != nil {
				return fmt.Errorf("validate watermark: %w", err)
			}
			err = pdfengines.EnsureStampFile(&stamp, stampFile)
			if err != nil {
				return fmt.Errorf("validate stamp: %w", err)
			}

			err = pdfengines.ValidatePdfFormatsCompat(pdfFormats, encrypt.UserPassword, embedPaths)
			if err != nil {
				return err
			}

			err = pdfengines.ValidatePdfEncryptCompat(encrypt)
			if err != nil {
				return err
			}

			err = pdfengines.ValidateFacturXCompat(facturX, facturxXmlPath, pdfFormats)
			if err != nil {
				return err
			}

			// Factur-X requires PDF/A-3; default to PDF/A-3b when no format was
			// requested. The conversion runs as a post-processing step below.
			pdfFormats = pdfengines.FacturXPdfFormats(ctx, engine, facturX, pdfFormats, true, nil)

			hasPostProcessing := watermark.Source != "" || stamp.Source != "" || angle != 0 ||
				len(embedPaths) > 0 || len(metadata) > 0 || flatten || facturX.ConformanceLevel != ""

			outputPaths := make([]string, len(inputPaths))
			for i, inputPath := range inputPaths {
				outputPaths[i] = ctx.GeneratePath(".pdf")
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
				}

				if nativePdfFormats && splitMode == zeroValuedSplitMode && !hasPostProcessing {
					// Only natively apply given PDF formats if we're not
					// splitting the PDF later and no post-processing features
					// are enabled (as they would degrade compliance).
					options.PdfFormats = pdfFormats
				}

				err = libreOffice.Pdf(ctx, ctx.Log(), inputPath, outputPaths[i], options)
				if err != nil {
					if errors.Is(err, libreofficeapi.ErrInvalidPdfFormats) {
						return api.WrapError(
							fmt.Errorf("convert to PDF: %w", err),
							api.NewSentinelHttpError(
								http.StatusBadRequest,
								fmt.Sprintf("The PDF format '%s' is not supported. Valid formats include PDF/A-1b, PDF/A-2b, PDF/A-3b, and PDF/UA.", pdfFormats.PdfA),
							),
						)
					}

					filename := ctx.OriginalFilename(inputPath)

					if errors.Is(err, libreofficeapi.ErrIoException) || errors.Is(err, libreofficeapi.ErrIllegalArgumentException) {
						return api.WrapError(
							fmt.Errorf("convert to PDF: %w", err),
							api.NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("LibreOffice could not read the document '%s'. Ensure the file is not corrupted and that its extension matches its actual format.", filename)),
						)
					}

					if errors.Is(err, libreofficeapi.ErrCannotConvertException) {
						return api.WrapError(
							fmt.Errorf("convert to PDF: %w", err),
							api.NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("LibreOffice read the document '%s' but could not convert it to PDF. The document may be corrupted or rely on an unsupported feature.", filename)),
						)
					}

					// Exit codes 5 and 6 name the UNO exception class that was
					// caught, not a cause: both cover a client mistake and a
					// LibreOffice crash. Blame the client only when one of its
					// inputs is actually implicated, since the server is the
					// only remaining explanation otherwise. Password evidence
					// outranks page ranges: a password failure aborts on import,
					// before the export filter applies any page range.
					// See https://github.com/gotenberg/gotenberg/issues/1588.
					if errors.Is(err, libreofficeapi.ErrUnoException) || errors.Is(err, libreofficeapi.ErrRuntimeException) {
						protection := libreofficeapi.DetectPasswordProtection(inputPath)

						var sentinel api.SentinelHttpError
						switch {
						case protection == libreofficeapi.PasswordProtectionRequired && options.Password == "":
							sentinel = api.NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("The document '%s' is password-protected. Provide its password in the 'password' form field.", filename))
						case protection == libreofficeapi.PasswordProtectionRequired:
							sentinel = api.NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("The password for the document '%s' is incorrect. Check the 'password' form field.", filename))
						case protection == libreofficeapi.PasswordProtectionNone && options.Password != "":
							sentinel = api.NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("The document '%s' is not password-protected. Remove the 'password' form field.", filename))
						case options.Password != "":
							sentinel = api.NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("LibreOffice could not open the document '%s' with the given password. Check the 'password' form field, and omit it if the document is not password-protected.", filename))
						case errors.Is(err, libreofficeapi.ErrUnoException) && options.PageRanges != "":
							sentinel = api.NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("LibreOffice could not apply the page ranges '%s' to the document '%s'. Check the 'nativePageRanges' form field; valid values look like '1-4', '2' or '1,3,5-7'.", options.PageRanges, filename))
						default:
							sentinel = api.NewSentinelHttpError(http.StatusInternalServerError, fmt.Sprintf(unattributableFailureMessage, filename))
						}

						return api.WrapError(fmt.Errorf("convert to PDF: %w", err), sentinel)
					}

					return fmt.Errorf("convert to PDF: %w", err)
				}
			}

			if merge {
				outputPath, err := pdfengines.MergeStub(ctx, engine, outputPaths)
				if err != nil {
					return fmt.Errorf("merge PDFs: %w", err)
				}

				// Only one output path.
				outputPaths = []string{outputPath}
			}

			if splitMode != zeroValuedSplitMode {
				if !merge {
					// document.docx -> document.docx.pdf, so that split naming
					// document.docx_0.pdf, etc.
					for i, inputPath := range inputPaths {
						originalName := ctx.OriginalFilename(inputPath)
						outputPath := ctx.GeneratePathFromFilename(originalName + ".pdf")

						err = ctx.Rename(outputPaths[i], outputPath)
						if err != nil {
							return fmt.Errorf("rename output path: %w", err)
						}

						outputPaths[i] = outputPath
					}
				}

				outputPaths, err = pdfengines.SplitPdfStub(ctx, engine, splitMode, outputPaths)
				if err != nil {
					return fmt.Errorf("split PDFs: %w", err)
				}
			}

			err = pdfengines.WatermarkStub(ctx, engine, watermark, outputPaths)
			if err != nil {
				return fmt.Errorf("watermark PDFs: %w", err)
			}

			err = pdfengines.StampStub(ctx, engine, stamp, outputPaths)
			if err != nil {
				return fmt.Errorf("stamp PDFs: %w", err)
			}

			err = pdfengines.RotateStub(ctx, engine, angle, rotatePages, outputPaths)
			if err != nil {
				return fmt.Errorf("rotate PDFs: %w", err)
			}

			if flatten {
				err = pdfengines.FlattenStub(ctx, engine, outputPaths)
				if err != nil {
					return fmt.Errorf("flatten PDFs: %w", err)
				}
			}

			err = pdfengines.OptimizeStub(ctx, engine, optimizeImages, imageQuality, outputPaths)
			if err != nil {
				return fmt.Errorf("optimize PDF images: %w", err)
			}

			needsConvertStub := !nativePdfFormats ||
				(nativePdfFormats && splitMode != zeroValuedSplitMode) ||
				(nativePdfFormats && hasPostProcessing)
			if needsConvertStub {
				convertOutputPaths, err := pdfengines.ConvertStub(ctx, engine, pdfFormats, outputPaths)
				if err != nil {
					return fmt.Errorf("convert PDFs: %w", err)
				}

				if splitMode != zeroValuedSplitMode {
					// The PDF has been split and split parts have been converted to
					// specific formats. We want to keep the split naming.
					for i, convertOutputPath := range convertOutputPaths {
						err = ctx.Rename(convertOutputPath, outputPaths[i])
						if err != nil {
							return fmt.Errorf("rename output path: %w", err)
						}
					}
				} else {
					outputPaths = convertOutputPaths
				}
			}

			// Metadata, embeds are written after Convert, as LibreOffice
			// strips them during PDF/A conversion.
			err = pdfengines.WriteMetadataStub(ctx, engine, metadata, outputPaths)
			if err != nil {
				return fmt.Errorf("write metadata: %w", err)
			}

			err = pdfengines.EmbedFilesStub(ctx, engine, embedPaths, outputPaths)
			if err != nil {
				return fmt.Errorf("embed files into PDFs: %w", err)
			}

			err = pdfengines.EmbedFilesMetadataStub(ctx, engine, embedsMetadata, outputPaths)
			if err != nil {
				return fmt.Errorf("set embeds metadata: %w", err)
			}

			err = pdfengines.ApplyFacturXStub(ctx, engine, facturX, facturxXmlPath, outputPaths)
			if err != nil {
				return fmt.Errorf("apply Factur-X: %w", err)
			}

			err = pdfengines.EncryptPdfStub(ctx, engine, encrypt, outputPaths)
			if err != nil {
				return fmt.Errorf("encrypt PDFs: %w", err)
			}

			if len(outputPaths) > 1 && splitMode == zeroValuedSplitMode {
				// If .zip archive, document.docx -> document.docx.pdf.
				for i, inputPath := range inputPaths {
					originalName := ctx.OriginalFilename(inputPath)
					outputPath := ctx.GeneratePathFromFilename(originalName + ".pdf")

					err = ctx.Rename(outputPaths[i], outputPath)
					if err != nil {
						return fmt.Errorf("rename output path: %w", err)
					}

					outputPaths[i] = outputPath
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
