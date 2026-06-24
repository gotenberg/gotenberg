package poppler

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

// defaultDpi is the resolution used when the client omits the dpi field.
const defaultDpi = 150

// imageExtension maps an [ImageConvertOptions.Format] to the file extension
// used for the response. JPEG and TIFF reuse the format name for consistency
// with the Chromium screenshot routes, even though pdftoppm writes .jpg/.tif.
func imageExtension(format string) string {
	switch format {
	case "jpeg":
		return "jpeg"
	case "tiff":
		return "tiff"
	default:
		return "png"
	}
}

// FormDataPopplerImageOptions reads the [ImageConvertOptions] from the form
// data.
func FormDataPopplerImageOptions(form *api.FormData) ImageConvertOptions {
	options := ImageConvertOptions{
		Format:  "png",
		Dpi:     defaultDpi,
		Quality: 100,
	}

	form.
		Custom("format", func(value string) error {
			if value == "" {
				return nil
			}
			if value != "png" && value != "jpeg" && value != "tiff" {
				return errors.New("wrong value, expected either 'png', 'jpeg' or 'tiff'")
			}
			options.Format = value
			return nil
		}).
		Custom("dpi", func(value string) error {
			if value == "" {
				return nil
			}
			v, err := strconv.Atoi(value)
			if err != nil {
				return err
			}
			if v <= 0 {
				return errors.New("value must be a positive integer")
			}
			options.Dpi = v
			return nil
		}).
		Custom("quality", func(value string) error {
			if value == "" {
				return nil
			}
			v, err := strconv.Atoi(value)
			if err != nil {
				return err
			}
			if v < 1 || v > 100 {
				return errors.New("wrong value, expected an integer between 1 and 100")
			}
			options.Quality = v
			return nil
		}).
		Int("firstPage", &options.FirstPage, 0).
		Int("lastPage", &options.LastPage, 0)

	return options
}

// convertImageRoute returns an [api.Route] which rasterizes PDFs into images.
func convertImageRoute(engine *Poppler) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/pdfengines/convert/image",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)

			form := ctx.FormData()
			options := FormDataPopplerImageOptions(form)

			var inputPaths []string
			err := form.
				MandatoryPaths([]string{".pdf"}, &inputPaths).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			ext := imageExtension(options.Format)

			var outputPaths []string
			for _, inputPath := range inputPaths {
				originalName := ctx.OriginalFilename(inputPath)
				base := strings.TrimSuffix(originalName, filepath.Ext(originalName))

				outputDirPath, err := ctx.CreateSubDirectory(uuid.New().String())
				if err != nil {
					return fmt.Errorf("create sub-directory: %w", err)
				}

				paths, err := engine.Rasterize(ctx, ctx.Log(), inputPath, outputDirPath, options)
				if err != nil {
					return fmt.Errorf("rasterize PDF '%s': %w", originalName, err)
				}

				// Rename each page to a UUID path carrying the requested
				// extension so the response Content-Type is correct, and keep a
				// human-readable original filename for the archive entry.
				for i, path := range paths {
					newPath := fmt.Sprintf("%s/%s.%s", outputDirPath, uuid.New().String(), ext)
					err = ctx.Rename(path, newPath)
					if err != nil {
						return fmt.Errorf("rename image: %w", err)
					}

					ctx.RegisterDiskPath(newPath, fmt.Sprintf("%s-%d.%s", base, i+1, ext))
					outputPaths = append(outputPaths, newPath)
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
