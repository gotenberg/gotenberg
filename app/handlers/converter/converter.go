package converter

import (
	"fmt"
	"net/http"
	"os"

	gfile "github.com/gulien/gotenberg/app/handlers/converter/file"
	"github.com/gulien/gotenberg/app/handlers/converter/process"
	ghttp "github.com/gulien/gotenberg/app/handlers/http"

	"github.com/satori/go.uuid"
)

type Converter struct {
	files      []*gfile.File
	workingDir string
}

type NoFileToConvertError struct{}

func (e *NoFileToConvertError) Error() string {
	return "There is no file to convert"
}

func NewConverter(r *http.Request, contentType ghttp.ContentType) (*Converter, error) {
	c := &Converter{
		workingDir: fmt.Sprintf("./%s/", uuid.NewV4().String()),
	}

	if err := os.Mkdir(c.workingDir, 0666); err != nil {
		return nil, err
	}

	switch contentType {
	case ghttp.MultipartFormDataContentType:
		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			return nil, err
		}

		formData := r.MultipartForm
		files, ok := formData.File["files"]
		if !ok {
			return nil, nil
		}

		for i := range files {
			file, err := files[i].Open()
			if err != nil {
				return nil, err
			}

			defer file.Close()

			f, err := gfile.NewFile(c.workingDir, file)
			if err != nil {
				return nil, err
			}

			c.files = append(c.files, f)
		}
		break
	default:
		f, err := gfile.NewFile(c.workingDir, r.Body)
		if err != nil {
			return nil, err
		}

		c.files = append(c.files, f)
	}

	if len(c.files) == 0 {
		return nil, &NoFileToConvertError{}
	}

	return c, nil
}

func (c *Converter) Convert() (string, error) {
	var filesPaths []string
	for _, f := range c.files {
		if f.Type != gfile.PDFType {
			path, err := process.ExecConversion(c.workingDir, f)
			if err != nil {
				return "", err
			}

			filesPaths = append(filesPaths, path)
		} else {
			filesPaths = append(filesPaths, f.Path)
		}
	}

	if len(filesPaths) == 1 {
		return filesPaths[0], nil
	}

	path, err := process.ExecMerge(c.workingDir, filesPaths)
	if err != nil {
		return "", err
	}

	return path, nil
}

func (c *Converter) Clear() error {
	if err := os.RemoveAll(c.workingDir); err != nil {
		return err
	}

	return nil
}
