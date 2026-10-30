package api

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
)

// ArchivesFormField is the form field whose zip archives the [Context]
// unpacks: each entry becomes a form file, as if uploaded under "files". A zip
// archive uploaded under any other field stays a file.
const ArchivesFormField = "archives"

// errArchiveTooLarge reports an archive unpacking past its allowance.
var errArchiveTooLarge = errors.New("archive unpacks past its allowance")

// extractArchive unpacks the zip archive fh into the working directory. Its
// entries count against budget, and together may unpack to cfg.maxRatio times
// the archive's size. A directory, a symbolic link, and the metadata macOS
// adds are skipped. A nested archive is a file: it is not unpacked.
func (ctx *Context) extractArchive(fh *multipart.FileHeader, cfg archiveConfig, budget *bodyBudget) error {
	invalid := func(err error, reason string) error {
		return WrapError(err, NewSentinelHttpError(
			http.StatusBadRequest,
			fmt.Sprintf("Invalid form data: form field '%s' is invalid: '%s' %s", ArchivesFormField, sanitizeFilename(fh.Filename), reason),
		))
	}

	in, err := fh.Open()
	if err != nil {
		return fmt.Errorf("open multipart file: %w", err)
	}
	defer func() {
		err := in.Close()
		if err != nil {
			ctx.logger.ErrorContext(context.Background(), fmt.Sprintf("close file header: %s", err))
		}
	}()

	r, err := zip.NewReader(in, fh.Size)
	if err != nil {
		return invalid(err, "is not a zip archive")
	}

	if len(r.File) > cfg.maxEntries {
		return invalid(fmt.Errorf("%d entries", len(r.File)), fmt.Sprintf("holds more than %d entries", cfg.maxEntries))
	}

	allowance := &unpackAllowance{remaining: cfg.maxRatio * max(fh.Size, 1)}

	for _, f := range r.File {
		if !f.Mode().IsRegular() || isMacOSMetadata(f.Name) {
			continue
		}

		filename := sanitizeFilename(f.Name)
		if filename == "" {
			continue
		}

		// Bit 0 of the general purpose flags marks an encrypted entry
		// (APPNOTE.TXT, 4.4.4), which archive/zip cannot read.
		if f.Flags&0x1 != 0 {
			return invalid(fmt.Errorf("entry %q is encrypted", f.Name), "holds encrypted entries")
		}

		err = ctx.extractEntry(f, filename, allowance, budget)
		if errors.Is(err, errArchiveTooLarge) {
			return invalid(err, fmt.Sprintf("unpacks to more than %d times its size", cfg.maxRatio))
		}
		if errors.Is(err, zip.ErrFormat) || errors.Is(err, zip.ErrAlgorithm) || errors.Is(err, zip.ErrChecksum) {
			return invalid(err, fmt.Sprintf("has an unreadable entry %q", filename))
		}
		if err != nil {
			return fmt.Errorf("unpack entry %q: %w", f.Name, err)
		}
	}

	return nil
}

func (ctx *Context) extractEntry(f *zip.File, filename string, allowance *unpackAllowance, budget *bodyBudget) error {
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("open entry: %w", err)
	}
	defer func() {
		err := rc.Close()
		if err != nil {
			ctx.logger.ErrorContext(context.Background(), fmt.Sprintf("close archive entry: %s", err))
		}
	}()

	// The bytes unpacked count, not the size the entry header declares.
	path, err := ctx.writeToDisk(filename, &trackingReader{R: rc, AddReadBytes: allowance.take}, budget)
	if err != nil {
		return err
	}

	ctx.addFile(filename, path)

	return nil
}

// isMacOSMetadata reports whether an entry is metadata the macOS archiver
// adds: the __MACOSX directory and the ._ resource forks.
func isMacOSMetadata(name string) bool {
	return strings.HasPrefix(name, "__MACOSX/") || strings.HasPrefix(filepath.Base(name), "._")
}

// unpackAllowance counts down the bytes an archive may still unpack to.
type unpackAllowance struct {
	remaining int64
}

func (a *unpackAllowance) take(n int64) error {
	a.remaining -= n
	if a.remaining < 0 {
		return errArchiveTooLarge
	}

	return nil
}

// addFile registers the file at path as an uploaded file named filename.
func (ctx *Context) addFile(filename, path string) {
	unique := ctx.uniqueFilename(filename)
	ctx.files[unique] = path
	ctx.diskToOriginal[path] = unique
	ctx.trackFileOrder(path, filename)
}
