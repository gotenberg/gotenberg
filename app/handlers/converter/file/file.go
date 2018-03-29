package file

import (
	"fmt"
	"io"
	"os"

	ghttp "github.com/gulien/gotenberg/app/handlers/http"

	"github.com/satori/go.uuid"
)

type (
	File struct {
		Type FileType
		Path string
	}

	FileType string

	FileExt string
)

const (
	PDFType    FileType = "PDF"
	HTMLType   FileType = "HTML"
	OfficeType FileType = "Office"

	PDFExt    FileExt = ".pdf"
	HTMLExt   FileExt = ".html"
	OfficeExt FileExt = ""
)

func NewFile(r io.Reader) (*File, error) {
	f := &File{
		Path: MakeFilePath(),
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

	f, err = reworkFilePath(f)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func MakeFilePath() string {
	return fmt.Sprintf("./%s", uuid.NewV4().String())
}

var filesTypes = map[ghttp.ContentType]FileType{
	ghttp.PDFContentType:         PDFType,
	ghttp.HTMLContentType:        HTMLType,
	ghttp.OctetStreamContentType: OfficeType,
	ghttp.ZipContentType:         OfficeType,
}

func findFileType(f *os.File) (FileType, error) {
	ct, err := ghttp.SniffContentType(f)
	if err != nil {
		return "", err
	}

	t, ok := filesTypes[ct]
	if !ok {
		// TODO error
		return "", nil
	}

	return t, nil
}

var filesExtensions = map[FileType]FileExt{
	PDFType:    PDFExt,
	HTMLType:   HTMLExt,
	OfficeType: OfficeExt,
}

func reworkFilePath(f *File) (*File, error) {
	ext, ok := filesExtensions[f.Type]
	if !ok {
		// TODO error
		return nil, nil
	}

	if ext != OfficeExt {
		newPath := fmt.Sprintf("./%s%s", MakeFilePath(), ext)

		err := os.Rename(f.Path, newPath)
		if err != nil {
			return nil, err
		}

		f.Path = newPath
	}

	return f, nil
}
