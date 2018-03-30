package file

import (
	"fmt"
	"io"
	"os"

	ghttp "github.com/gulien/gotenberg/app/handlers/http"

	"github.com/satori/go.uuid"
)

type File struct {
	Type FileType
	Path string
}

type FileType uint32

const (
	PDFType FileType = iota
	HTMLType
	OfficeType
)

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

func MakeFilePath(workingDir string) string {
	return fmt.Sprintf("%s%s", workingDir, uuid.NewV4().String())
}

var filesTypes = map[ghttp.ContentType]FileType{
	ghttp.PDFContentType:         PDFType,
	ghttp.HTMLContentType:        HTMLType,
	ghttp.OctetStreamContentType: OfficeType,
	ghttp.ZipContentType:         OfficeType,
}

type fileTypeNotFound struct{}

func (e *fileTypeNotFound) Error() string {
	return "The file type was not found for the given 'Content-Type'"
}

func findFileType(f *os.File) (FileType, error) {
	ct, err := ghttp.SniffContentType(f)
	if err != nil {
		return 999, err
	}

	t, ok := filesTypes[ct]
	if !ok {
		return 999, &fileTypeNotFound{}
	}

	return t, nil
}

type FileExt string

const (
	PDFExt    FileExt = ".pdf"
	HTMLExt   FileExt = ".html"
	OfficeExt FileExt = ""
)

var filesExtensions = map[FileType]FileExt{
	PDFType:    PDFExt,
	HTMLType:   HTMLExt,
	OfficeType: OfficeExt,
}

type fileExtNotFound struct{}

func (e *fileExtNotFound) Error() string {
	return "The file extension was not found for the given file type"
}

func reworkFilePath(workingDir string, f *File) (*File, error) {
	ext, ok := filesExtensions[f.Type]
	if !ok {
		return nil, &fileExtNotFound{}
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
