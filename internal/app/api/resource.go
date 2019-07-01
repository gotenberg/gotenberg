package api

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
)

const (
	resultFilename string = "resultFilename"
	waitTimeout    string = "waitTimeout"
	webhookURL     string = "webhookURL"
	remoteURL      string = "remoteURL"
	waitDelay      string = "waitDelay"
	paperWidth     string = "paperWidth"
	paperHeight    string = "paperHeight"
	marginTop      string = "marginTop"
	marginBottom   string = "marginBottom"
	marginLeft     string = "marginLeft"
	marginRight    string = "marginRight"
	landscape      string = "landscape"
)

type resource struct {
	formValues       map[string]string
	formFilesDirPath string
}

func newResource(c echo.Context, dirPath string) (*resource, error) {
	r := &resource{
		formValues: formValues(c),
	}
	r.formFilesDirPath = dirPath
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, fmt.Errorf("%s: making directory: %v", dirPath, err)
	}
	if err := formFiles(c, dirPath); err != nil {
		return r, err
	}
	return r, nil
}

func formValues(c echo.Context) map[string]string {
	v := make(map[string]string)
	v[resultFilename] = c.FormValue(resultFilename)
	v[waitTimeout] = c.FormValue(waitTimeout)
	v[webhookURL] = c.FormValue(webhookURL)
	v[remoteURL] = c.FormValue(remoteURL)
	v[waitDelay] = c.FormValue(waitDelay)
	v[paperWidth] = c.FormValue(paperWidth)
	v[paperHeight] = c.FormValue(paperHeight)
	v[marginTop] = c.FormValue(marginTop)
	v[marginBottom] = c.FormValue(marginBottom)
	v[marginLeft] = c.FormValue(marginLeft)
	v[marginRight] = c.FormValue(marginRight)
	v[landscape] = c.FormValue(landscape)
	return v
}

func formFiles(c echo.Context, dirPath string) error {
	form, err := c.MultipartForm()
	if err != nil {
		return fmt.Errorf("getting multipart form: %v", err)
	}
	for _, files := range form.File {
		for _, fh := range files {
			in, err := fh.Open()
			if err != nil {
				return fmt.Errorf("%s: opening file: %v", fh.Filename, err)
			}
			defer in.Close() // nolint: errcheck
			fpath := fmt.Sprintf("%s/%s", dirPath, fh.Filename)
			out, err := os.Create(fpath)
			if err != nil {
				return fmt.Errorf("%s: creating new file: %v", fpath, err)
			}
			defer out.Close() // nolint: errcheck
			if err := out.Chmod(0644); err != nil {
				return fmt.Errorf("%s: changing file mode: %v", fpath, err)
			}
			if _, err := io.Copy(out, in); err != nil {
				return fmt.Errorf("%s: writing file: %v", fpath, err)
			}
			if _, err := out.Seek(0, 0); err != nil {
				return fmt.Errorf("%s: resetting read pointer: %v", fpath, err)
			}
		}
	}
	return nil
}

func (r *resource) close() error {
	if _, err := os.Stat(r.formFilesDirPath); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(r.formFilesDirPath)
}

const defaultHeaderFooterHTML string = "<html><head></head><body></body></html>"

func (r *resource) chromePrinterOptions(defaultWaitTimeout float64) (*printer.ChromeOptions, error) {
	timeout, err := r.float64(waitTimeout, defaultWaitTimeout)
	if err != nil {
		return nil, err
	}
	delay, err := r.float64(waitDelay, 0.0)
	if err != nil {
		return nil, err
	}
	header, err := r.content("header.html", defaultHeaderFooterHTML)
	if err != nil {
		return nil, err
	}
	footer, err := r.content("footer.html", defaultHeaderFooterHTML)
	if err != nil {
		return nil, err
	}
	width, err := r.float64(paperWidth, 8.27)
	if err != nil {
		return nil, err
	}
	height, err := r.float64(paperHeight, 11.7)
	if err != nil {
		return nil, err
	}
	top, err := r.float64(marginTop, 1)
	if err != nil {
		return nil, err
	}
	bottom, err := r.float64(marginBottom, 1)
	if err != nil {
		return nil, err
	}
	left, err := r.float64(marginLeft, 1)
	if err != nil {
		return nil, err
	}
	right, err := r.float64(marginRight, 1)
	if err != nil {
		return nil, err
	}
	landscape, err := r.bool(landscape, false)
	if err != nil {
		return nil, err
	}
	return &printer.ChromeOptions{
		WaitTimeout:  timeout,
		WaitDelay:    delay,
		HeaderHTML:   header,
		FooterHTML:   footer,
		PaperWidth:   width,
		PaperHeight:  height,
		MarginTop:    top,
		MarginBottom: bottom,
		MarginLeft:   left,
		MarginRight:  right,
		Landscape:    landscape,
	}, nil
}

func (r *resource) officePrinterOptions(defaultWaitTimeout float64) (*printer.OfficeOptions, error) {
	timeout, err := r.float64(waitTimeout, defaultWaitTimeout)
	if err != nil {
		return nil, err
	}
	landscape, err := r.bool(landscape, false)
	if err != nil {
		return nil, err
	}
	return &printer.OfficeOptions{
		WaitTimeout: timeout,
		Landscape:   landscape,
	}, nil
}

func (r *resource) mergePrinterOptions(defaultWaitTimeout float64) (*printer.MergeOptions, error) {
	timeout, err := r.float64(waitTimeout, defaultWaitTimeout)
	if err != nil {
		return nil, err
	}
	return &printer.MergeOptions{
		WaitTimeout: timeout,
	}, nil
}

func (r *resource) has(key string) bool {
	v, ok := r.formValues[key]
	if ok {
		ok = v != ""
	}
	return ok
}

func (r *resource) hasFile(filename string) bool {
	fpath := fmt.Sprintf("%s/%s", r.formFilesDirPath, filename)
	_, err := os.Stat(fpath)
	return !os.IsNotExist(err)
}

func (r *resource) get(key string) (string, error) {
	v, ok := r.formValues[key]
	if !ok {
		return "", fmt.Errorf("form value %s does not exist", key)
	}
	return v, nil
}

func (r *resource) float64(key string, defaultValue float64) (float64, error) {
	if !r.has(key) {
		return defaultValue, nil
	}
	v, err := r.get(key)
	if err != nil {
		return 0.0, err
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0.0, fmt.Errorf("form value %s: %v", key, err)
	}
	return f, nil
}

func (r *resource) bool(key string, defaultValue bool) (bool, error) {
	if !r.has(key) {
		return defaultValue, nil
	}
	v, err := r.get(key)
	if err != nil {
		return false, err
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("form value %s: %v", key, err)
	}
	return b, nil
}

func (r *resource) fpath(filename string) (string, error) {
	fpath := fmt.Sprintf("%s/%s", r.formFilesDirPath, filename)
	_, err := os.Stat(fpath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("%s: form file does not exist", filename)
	}
	absPath, err := filepath.Abs(fpath)
	if err != nil {
		return "", fmt.Errorf("%s: getting absolute path: %v", fpath, err)
	}
	return absPath, nil
}

func (r *resource) content(filename string, defaultValue string) (string, error) {
	if !r.hasFile(filename) {
		return defaultValue, nil
	}
	fpath, err := r.fpath(filename)
	if err != nil {
		return "", err
	}
	b, err := ioutil.ReadFile(fpath)
	if err != nil {
		return "", fmt.Errorf("%s: reading form file: %v", fpath, err)
	}
	return string(b), nil
}

func (r *resource) fpaths(exts ...string) ([]string, error) {
	var fpaths []string
	err := filepath.Walk(r.formFilesDirPath, func(path string, info os.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}
		fpath, err := r.fpath(info.Name())
		if err != nil {
			return err
		}
		for _, ext := range exts {
			if filepath.Ext(fpath) == ext {
				fpaths = append(fpaths, fpath)
				return nil
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(fpaths) == 0 {
		return nil, fmt.Errorf("no form files found for extensions: %v", exts)
	}
	return fpaths, nil
}
