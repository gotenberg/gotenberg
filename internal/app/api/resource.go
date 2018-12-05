package api

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/labstack/echo"
	"github.com/thecodingmachine/gotenberg/internal/pkg/rand"
)

const (
	webhookURL  string = "webhookURL"
	paperWidth  string = "paperWidth"
	paperHeight string = "paperHeight"
	landscape   string = "landscape"
)

// resource facilitates storing and accessing
// data from a multipart/form-data request.
type resource struct {
	values  map[string]string
	dirPath string
}

func newResource(c echo.Context) (*resource, error) {
	v := make(map[string]string)
	v[webhookURL] = c.FormValue(webhookURL)
	v[paperWidth] = c.FormValue(paperWidth)
	v[paperHeight] = c.FormValue(paperHeight)
	v[landscape] = c.FormValue(landscape)
	dirPath, err := rand.Get()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, fmt.Errorf("%s: making directory: %v", dirPath, err)
	}
	r := &resource{values: v, dirPath: dirPath}
	form, err := c.MultipartForm()
	if err != nil {
		return nil, fmt.Errorf("getting multipart form: %v", err)
	}
	for _, files := range form.File {
		for _, fh := range files {
			in, err := fh.Open()
			if err != nil {
				return nil, fmt.Errorf("%s: opening file: %v", fh.Filename, err)
			}
			defer in.Close()
			if err := r.writeFile(fh.Filename, in); err != nil {
				return nil, err
			}
		}
	}
	return r, nil
}

func (r *resource) writeFile(filename string, in io.Reader) error {
	fpath := fmt.Sprintf("%s/%s", r.dirPath, filename)
	out, err := os.Create(fpath)
	if err != nil {
		return fmt.Errorf("%s: creating new file: %v", fpath, err)
	}
	defer out.Close()
	if err := out.Chmod(0644); err != nil {
		return fmt.Errorf("%s: changing file mode: %v", fpath, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("%s: writing file: %v", fpath, err)
	}
	if _, err := out.Seek(0, 0); err != nil {
		return fmt.Errorf("%s: resetting read pointer: %v", fpath, err)
	}
	return nil
}

func (r *resource) filePath(filename string) (string, error) {
	fpath := fmt.Sprintf("%s/%s", r.dirPath, filename)
	_, err := os.Stat(fpath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("%s: file does not exist", filename)
	}
	absPath, err := filepath.Abs(fpath)
	if err != nil {
		return "", fmt.Errorf("%s: getting absolute path: %v", fpath, err)
	}
	return absPath, nil
}

func (r *resource) filePaths(exts []string) ([]string, error) {
	var fpaths []string
	filepath.Walk(r.dirPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		fpath, err := r.filePath(info.Name())
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
	return fpaths, nil
}

func (r *resource) paperSize() ([2]float64, error) {
	defaultSize := [2]float64{8.27, 11.7}
	widthStr := r.values[paperWidth]
	heightStr := r.values[paperHeight]
	if widthStr == "" || heightStr == "" {
		return defaultSize, nil
	}
	width, err := strconv.ParseFloat(widthStr, 64)
	if err != nil {
		return defaultSize, fmt.Errorf("paper width: %v", err)
	}
	height, err := strconv.ParseFloat(heightStr, 64)
	if err != nil {
		return defaultSize, fmt.Errorf("paper height: %v", err)
	}
	return [2]float64{width, height}, nil
}

func (r *resource) landscape() (bool, error) {
	landscapeStr := r.values[landscape]
	if landscapeStr == "" {
		return false, nil
	}
	landscape, err := strconv.ParseBool(landscapeStr)
	if err != nil {
		return false, fmt.Errorf("landscape: %v", err)
	}
	return landscape, nil
}

func (r *resource) webhookURL() string { return r.values[webhookURL] }
func (r *resource) removeAll() error   { return os.RemoveAll(r.dirPath) }
