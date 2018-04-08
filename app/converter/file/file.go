// Package file implements a solution for handling files coming from a request.
package file

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/satori/go.uuid"
)

// File represents a file which has been created
// from a request.
type File struct {
	// Type is the kind of file.
	Type Type
	// Path is the file path.
	Path string
}

// Type represents what kind of file we're dealing with.
type Type uint32

const (
	// PDFType represents a... PDF file.
	PDFType Type = iota
	// HTMLType represents an... HTML file.
	HTMLType
	// OfficeType represents an... Office document.
	OfficeType
)

// filesTypes associates a file extension with its file kind counterpart.
var filesTypes = map[string]Type{
	".pdf":  PDFType,
	".html": HTMLType,
	".doc":  OfficeType,
	".docx": OfficeType,
	".odt":  OfficeType,
	".xls":  OfficeType,
	".xlsx": OfficeType,
	".ods":  OfficeType,
	".ppt":  OfficeType,
	".pptx": OfficeType,
	".odp":  OfficeType,
}

type fileTypeNotFoundError struct {
	fileName string
}

func (e *fileTypeNotFoundError) Error() string {
	return fmt.Sprintf("File type was not found for '%s'", e.fileName)
}

// NewFile creates a file in the considered directory.
// Returns a *File instance or an error if something bad happened.
func NewFile(workingDir string, r io.Reader, fileName string) (*File, error) {
	ext := filepath.Ext(fileName)

	t, ok := filesTypes[ext]
	if !ok {
		return nil, &fileTypeNotFoundError{fileName: fileName}
	}

	f := &File{
		Path: MakeFilePath(workingDir, ext),
		Type: t,
	}

	file, err := os.Create(f.Path)
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

	return f, nil
}

// MakeFilePath is a simple helper which generates a random file name
// and associates it with the considered directory to make a path.
func MakeFilePath(workingDir string, ext string) string {
	return fmt.Sprintf("%s%s%s", workingDir, uuid.NewV4().String(), ext)
}
