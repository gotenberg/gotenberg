package scenario

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/mholt/archives"
)

// recordResponse reads resp, closes its body, and returns it as a recorder
// along with its body.
func recordResponse(resp *http.Response) (*httptest.ResponseRecorder, []byte, error) {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read response body: %w", err)
	}

	rec := httptest.NewRecorder()
	rec.Code = resp.StatusCode
	for key, values := range resp.Header {
		for _, v := range values {
			rec.Header().Add(key, v)
		}
	}

	_, err = rec.Body.Write(body)
	if err != nil {
		return nil, nil, fmt.Errorf("write response body: %w", err)
	}

	return rec, body, nil
}

// outputFilename returns the filename a Content-Disposition header names, and
// false when it names none.
func outputFilename(contentDisposition string) (string, bool, error) {
	if contentDisposition == "" {
		return "", false, nil
	}

	_, params, err := mime.ParseMediaType(contentDisposition)
	if err != nil {
		return "", false, fmt.Errorf("parse Content-Disposition header: %w", err)
	}

	filename, ok := params["filename"]
	return filename, ok, nil
}

// storeOutput writes an output Gotenberg sent into dirPath, under filename,
// and extracts it there too when it is a zip archive.
func storeOutput(ctx context.Context, dirPath, filename, contentType string, body []byte) error {
	err := os.MkdirAll(dirPath, 0o755)
	if err != nil {
		return fmt.Errorf("create directory %q: %w", dirPath, err)
	}

	fpath := fmt.Sprintf("%s/%s", dirPath, filename)
	// #nosec
	file, err := os.Create(fpath)
	if err != nil {
		return fmt.Errorf("create file %q: %w", fpath, err)
	}
	defer file.Close()

	_, err = file.Write(body)
	if err != nil {
		return fmt.Errorf("write file %q: %w", fpath, err)
	}

	if contentType != "application/zip" {
		return nil
	}

	var format archives.Zip
	return format.Extract(ctx, file, func(ctx context.Context, f archives.FileInfo) error {
		source, err := f.Open()
		if err != nil {
			return fmt.Errorf("open file %q: %w", f.Name(), err)
		}
		defer source.Close()

		targetPath := fmt.Sprintf("%s/%s", dirPath, f.Name())
		// #nosec
		target, err := os.Create(targetPath)
		if err != nil {
			return fmt.Errorf("create file %q: %w", targetPath, err)
		}
		defer target.Close()

		_, err = io.Copy(target, source)
		if err != nil {
			return fmt.Errorf("copy file %q: %w", targetPath, err)
		}

		return nil
	})
}
