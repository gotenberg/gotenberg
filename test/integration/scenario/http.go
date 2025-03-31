package scenario

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

func doRequest(method, url string, headers map[string]string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create a request: %w", err)
	}

	for header, value := range headers {
		req.Header.Set(header, value)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send a request: %w", err)
	}

	return resp, nil
}

func doFormDataRequest(method, url string, fields map[string]string, files map[string][]string, headers map[string]string) (*http.Response, error) {
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	for name, value := range fields {
		err := writer.WriteField(name, value)
		if err != nil {
			return nil, fmt.Errorf("write field %q: %w", name, err)
		}
	}

	for name, paths := range files {
		for _, path := range paths {
			part, err := writer.CreateFormFile(name, filepath.Base(path))
			if err != nil {
				return nil, fmt.Errorf("create form file %q: %w", filepath.Base(path), err)
			}

			reader, err := os.Open(path)
			if err != nil {
				return nil, fmt.Errorf("open file %q: %w", path, err)
			}
			defer reader.Close()

			_, err = io.Copy(part, reader)
			if err != nil {
				return nil, fmt.Errorf("copy file %q: %w", path, err)
			}
		}
	}

	err := writer.Close()
	if err != nil {
		return nil, fmt.Errorf("close writer: %w", err)
	}

	req, err := http.NewRequest(method, url, &b)
	if err != nil {
		return nil, fmt.Errorf("create a request: %w", err)
	}

	for header, value := range headers {
		req.Header.Set(header, value)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send a request: %w", err)
	}

	return resp, nil
}
