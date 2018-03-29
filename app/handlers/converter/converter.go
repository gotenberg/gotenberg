package converter

import (
	"net/http"
	"os"

	gfile "github.com/gulien/gotenberg/app/handlers/converter/file"
	"github.com/gulien/gotenberg/app/handlers/converter/process"
	ghttp "github.com/gulien/gotenberg/app/handlers/http"
)

type Converter struct {
	files            []*gfile.File
	resultFilesPaths []string
	FinalFilePath    string
}

func NewConverter(r *http.Request, contentType ghttp.ContentType) (*Converter, error) {
	c := &Converter{}

	switch contentType {
	case ghttp.MultipartFormDataContentType:
		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			return nil, err
		}

		formData := r.MultipartForm
		files, ok := formData.File["files"]
		if !ok {
			// TODO
			return nil, nil
		}

		for i := range files {
			file, err := files[i].Open()
			if err != nil {
				return nil, err
			}

			defer file.Close()

			f, err := gfile.NewFile(file)
			if err != nil {
				// todo err
				return nil, err
			}

			c.files = append(c.files, f)
		}
		break
	default:
		f, err := gfile.NewFile(r.Body)
		if err != nil {
			// todo err
			return nil, err
		}

		c.files = append(c.files, f)
	}

	return c, nil
}

func (c *Converter) Convert() error {
	if len(c.files) == 0 {
		return &noFileToConvertError{}
	}

	var filesPaths []string
	for _, f := range c.files {
		if f.Type != gfile.PDFType {
			path, err := process.ExecConversion(f)
			if err != nil {
				return err
			}

			filesPaths = append(filesPaths, path)
		} else {
			filesPaths = append(filesPaths, f.Path)
		}
	}

	path, err := process.ExecMerge(filesPaths)
	if err != nil {
		return err
	}

	c.resultFilesPaths = filesPaths
	c.FinalFilePath = path

	return nil
}

func (c *Converter) Clear() error {
	for _, f := range c.files {
		err := os.Remove(f.Path)
		if err != nil {
			return err
		}
	}

	for _, path := range c.resultFilesPaths {
		err := os.Remove(path)
		if err != nil {
			return err
		}
	}

	if c.FinalFilePath != "" {
		err := os.Remove(c.FinalFilePath)
		if err != nil {
			return err
		}
	}

	return nil
}
