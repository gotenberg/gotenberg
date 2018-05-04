// Package file implements a solution for handling files coming from a request.
package file

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/thecodingmachine/gotenberg/app/config"
	"github.com/thecodingmachine/gotenberg/app/logger"

	"github.com/dustin/go-humanize"
	"github.com/satori/go.uuid"
)

// File represents a file which has been created
// from a request.
type File struct {
	// Extension is the extension of the file.
	Extension string
	// Path is the file path.
	Path string
}

// NewFile creates a file in the considered directory.
// Returns a *File instance or an error if something bad happened.
func NewFile(workingDir string, r io.Reader, fileName string) (*File, error) {
	ext := filepath.Ext(fileName)

	if _, err := config.GetCommand(ext); err != nil {
		return nil, err
	}

	f := &File{ext, MakeFilePath(workingDir, ext)}

	file, err := os.Create(f.Path)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	n, err := io.Copy(file, r)
	if err != nil {
		return nil, err
	}

	// resets the read pointer.
	file.Seek(0, 0)

	logger.Debugf("working file %s has been created from %s (%s copied)", f.Path, fileName, humanize.Bytes(uint64(n)))
	return f, nil
}

// MakeFilePath is a simple helper which generates a random file name
// and associates it with the considered directory to make a path.
func MakeFilePath(workingDir string, ext string) string {
	return fmt.Sprintf("%s%s%s", workingDir, uuid.NewV4().String(), ext)
}
