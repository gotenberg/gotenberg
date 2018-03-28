// Package converters implements a solution for reading files from a request
// and converting them to PDF.
package converters

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"text/template"

	"github.com/gulien/gotenberg/config"
	"github.com/gulien/gotenberg/helpers"

	"github.com/satori/go.uuid"
)

type (
	// Converter is the interface used in our convertHandler middleware.
	// Indeed, we don't want to know in this middleware which converter is actually used.
	Converter interface {
		// Convert should returns the file path of the PDF created by the converter.
		Convert() (string, error)
		// Clear should removes all files used by the converter.
		Clear() error
	}

	// DirectConverter allows us to convert a file to PDF.
	DirectConverter struct {
		// contentType is the content type of the file to convert.
		contentType string
		// filePath is the path of the file to convert.
		filePath string
		// resultFilePath is the path of the PDF created by
		// the conversion.
		resultFilePath string
	}

	// MultipartFormDataConverter contains an array of
	// DirectConverter instances, each one having to convert a file to PDF.
	MultipartFormDataConverter struct {
		// converters contains all DirectConverter instances which will be used
		// to convert files to PDF.
		converters []*DirectConverter
		// resultFilePath is the path of the PDF file created by the merge
		// of all PDF files created by the DirectConverter instances.
		resultFilePath string
	}
)

// NewConverter instantiates a converter according to the content type of the request.
func NewConverter(contentType string, r *http.Request) (Converter, error) {
	switch contentType {
	case "multipart/form-data":
		return newMultipartFromDataConverter(contentType, r)
	default:
		return newDirectConverter(contentType, r.Body)
	}
}

// ConverterUnprocessableEntityError is a custom error which is throwed when
// a file's content type does not match with one of the allowed content types.
type ConverterUnprocessableEntityError struct {
	message string
}

// Error is the implementation of the Error function from the error interface.
func (e *ConverterUnprocessableEntityError) Error() string {
	return e.message
}

// filesExtensions associates all allowed content types with their file extension.
var filesExtensions = map[string]string{
	"application/pdf":          ".pdf",
	"text/html":                ".html",
	"application/octet-stream": ".doc",
	"application/msword":       ".doc",
	"application/zip":          ".docx",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": ".docx",
}

// newDirectConverter instantiates a DirectConverter.
func newDirectConverter(contentType string, r io.Reader) (*DirectConverter, error) {
	fileExtension, ok := filesExtensions[contentType]
	if !ok {
		return nil, &ConverterUnprocessableEntityError{message: "No file extension found"}
	}

	filePath := fmt.Sprintf("./%s%s", uuid.NewV4().String(), fileExtension)

	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	_, err = io.Copy(file, r)
	if err != nil {
		return nil, err
	}
	// resets the read pointer.
	file.Seek(0, 0)

	contentType, err = helpers.DetectFileContentType(file)
	if err != nil {
		return nil, &ConverterUnprocessableEntityError{message: fmt.Sprintf("An error occured while trying to detect the content type of a file: %s", err.Error())}
	}

	return &DirectConverter{contentType: contentType, filePath: filePath}, nil
}

// ConversionCommandData will be applied to the data-driven command's template
// which will convert a file to PDF.
type ConversionCommandData struct {
	// FilePath is the path of the file to convert to PDF.
	FilePath string
	// The path of the PDF file created by the considered command.
	ResultFilePath string
}

// Convert converts a file to PDF.
func (c *DirectConverter) Convert() (string, error) {
	var cmdTemplate *template.Template

	switch c.contentType {
	case "text/html":
		cmdTemplate = config.AppConfig.Templates.HTMLtoPDF
		break
	case "application/octet-stream", "application/msword", "application/zip", "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		cmdTemplate = config.AppConfig.Templates.WordToPDF
		break
	default:
		// case "application/pdf".
		cmdTemplate = nil
		c.resultFilePath = c.filePath

		return c.resultFilePath, nil
	}

	c.resultFilePath = fmt.Sprintf("./%s.pdf", uuid.NewV4().String())

	cmdData := &ConversionCommandData{
		FilePath:       c.filePath,
		ResultFilePath: c.resultFilePath,
	}

	var data bytes.Buffer
	if err := cmdTemplate.Execute(&data, cmdData); err != nil {
		return "", fmt.Errorf("An error occured while executing a template: %s", err)
	}
	cmd := data.String()

	e := exec.Command("/bin/sh", "-c", cmd)
	if err := e.Run(); err != nil {
		return "", fmt.Errorf("An error occured while executing the command %s: %s", cmd, err)
	}

	return c.resultFilePath, nil
}

// Clear removes all files used by an instance of DirectConverter.
func (c *DirectConverter) Clear() error {
	if err := os.Remove(c.filePath); err != nil {
		return err
	}

	// if "application/pdf" content type, the file path is the same
	// as the result file path.
	if c.contentType != "application/pdf" {
		if err := os.Remove(c.resultFilePath); err != nil {
			return err
		}
	}

	return nil
}

// newMultipartFromDataConverter instantiates a MultipartFormDataConverter.
func newMultipartFromDataConverter(contentType string, r *http.Request) (*MultipartFormDataConverter, error) {
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		return nil, err
	}

	formData := r.MultipartForm
	files := formData.File["files"]
	c := &MultipartFormDataConverter{}

	for i := range files {
		file, err := files[i].Open()
		if err != nil {
			return nil, err
		}

		defer file.Close()

		contentType, err := helpers.DetectMultipartFileContentType(file)
		if err != nil {
			return nil, err
		}

		d, err := newDirectConverter(contentType, file)
		if err != nil {
			return nil, err
		}

		c.converters = append(c.converters, d)
	}

	return c, nil
}

// MergeCommandData will be applied to the data-driven command's template
// which will merge PDF files.
type MergeCommandData struct {
	// FilesPaths are the paths of the PDF files to merge.
	FilesPaths []string
	// The path of the PDF file created by the considered command.
	ResultFilePath string
}

// Convert converts all files from form data to PDF
// and then merges those resulting PDF into one final PDF.
func (c *MultipartFormDataConverter) Convert() (string, error) {
	var filesPaths []string

	for _, d := range c.converters {
		filePath, err := d.Convert()
		if err != nil {
			return "", err
		}

		filesPaths = append(filesPaths, filePath)
	}

	c.resultFilePath = fmt.Sprintf("./%s.pdf", uuid.NewV4().String())
	cmdTemplate := config.AppConfig.Templates.MergePDF
	cmdData := &MergeCommandData{
		FilesPaths:     filesPaths,
		ResultFilePath: c.resultFilePath,
	}

	var data bytes.Buffer
	if err := cmdTemplate.Execute(&data, cmdData); err != nil {
		return "", fmt.Errorf("An error occured while executing a template: %s", err)
	}
	cmd := data.String()

	e := exec.Command("/bin/sh", "-c", cmd)
	if err := e.Run(); err != nil {
		return "", fmt.Errorf("An error occured while executing the command %s: %s", cmd, err)
	}

	return c.resultFilePath, nil
}

// Clear removes all files used by an instance of MultipartFormDataConverter.
func (c *MultipartFormDataConverter) Clear() error {
	for _, d := range c.converters {
		if err := d.Clear(); err != nil {
			return err
		}
	}

	if err := os.Remove(c.resultFilePath); err != nil {
		return err
	}

	return nil
}
