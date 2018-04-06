// Package file implements a solution for handling files coming from a request.
package file

import (
	"fmt"
	"io"
	"os"

	ghttp "github.com/thecodingmachine/gotenberg/app/http"

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

// NewFile creates a file in the considered directory.
// Returns a *File instance or an error if something bad happened.
func NewFile(workingDir string, r io.Reader) (*File, error) {
	f := &File{
		Path: MakeFilePath(workingDir),
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

	t, err := findFileType(file)
	if err != nil {
		return nil, err
	}

	f.Type = t

	f, err = reworkFilePath(workingDir, f)
	if err != nil {
		return nil, err
	}

	return f, nil
}

// MakeFilePath is a simple helper which generates a random file name
// and associates it with the considered directory to make a path.
func MakeFilePath(workingDir string) string {
	return fmt.Sprintf("%s%s", workingDir, uuid.NewV4().String())
}

// filesTypes associates a content type with its file kind counterpart.
var filesTypes = map[ghttp.ContentType]Type{
	ghttp.PDFContentType:         PDFType,
	ghttp.HTMLContentType:        HTMLType,
	ghttp.OctetStreamContentType: OfficeType,
	ghttp.ZipContentType:         OfficeType,
}

type fileTypeNotFoundError struct{}

const fileTypeNotFoundErrorMessage = "The file type was not found for the given 'Content-Type'"

func (e *fileTypeNotFoundError) Error() string {
	return fileTypeNotFoundErrorMessage
}

// findFileType tries to detect what kind of file is the given file.
func findFileType(f *os.File) (Type, error) {
	ct, err := ghttp.SniffContentType(f)
	if err != nil {
		return 999, err
	}

	t, ok := filesTypes[ct]
	if !ok {
		return 999, &fileTypeNotFoundError{}
	}

	return t, nil
}

// Ext represents a file extension.
type Ext string

const (
	// PDFExt represents a... PDF extension.
	PDFExt Ext = ".pdf"
	// HTMLExt represents an... HTML extension.
	HTMLExt Ext = ".html"
	// OfficeExt is a empty string, as Office documents
	// have a lot of different extensions (.docx, .doc and so on).
	OfficeExt Ext = ""
)

// filesExtensions associates a kind of file with its extension.
var filesExtensions = map[Type]Ext{
	PDFType:    PDFExt,
	HTMLType:   HTMLExt,
	OfficeType: OfficeExt,
}

type fileExtNotFoundError struct{}

const fileExtNotFoundErrorMessage = "The file extension was not found for the given file type"

func (e *fileExtNotFoundError) Error() string {
	return fileExtNotFoundErrorMessage
}

// reworkFilePath renames a file in the considered directory and adds its extension.
func reworkFilePath(workingDir string, f *File) (*File, error) {
	ext, ok := filesExtensions[f.Type]
	if !ok {
		return nil, &fileExtNotFoundError{}
	}

	if ext != OfficeExt {
		newPath := fmt.Sprintf("%s%s", MakeFilePath(workingDir), ext)

		err := os.Rename(f.Path, newPath)
		if err != nil {
			return nil, err
		}

		f.Path = newPath
	}

	return f, nil
}
