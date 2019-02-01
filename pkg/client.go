package gotenberg

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

const (
	remoteURL       string = "remoteURL"
	webhookURL      string = "webhookURL"
	paperWidth      string = "paperWidth"
	paperHeight     string = "paperHeight"
	marginTop       string = "marginTop"
	marginBottom    string = "marginBottom"
	marginLeft      string = "marginLeft"
	marginRight     string = "marginRight"
	landscape       string = "landscape"
	webFontsTimeout string = "webFontsTimeout"
)

var (
	// A3 paper size.
	A3 = [2]float64{11.7, 16.5}
	// A4 paper size.
	A4 = [2]float64{8.27, 11.7}
	// A5 paper size.
	A5 = [2]float64{5.8, 8.3}
	// A6 paper size.
	A6 = [2]float64{4.1, 5.8}
	// Letter paper size.
	Letter = [2]float64{8.5, 11}
	// Legal paper size.
	Legal = [2]float64{8.5, 14}
	// Tabloid paper size.
	Tabloid = [2]float64{11, 17}
)

var (
	// NoMargins removes margins.
	NoMargins = [4]float64{0, 0, 0, 0}
	// NormalMargins uses 1 inche margins.
	NormalMargins = [4]float64{1, 1, 1, 1}
	// LargeMargins uses 2 inche margins.
	LargeMargins = [4]float64{2, 2, 2, 2}
)

// Client facilitates interacting with
// the Gotenberg API.
type Client struct {
	Hostname string
}

// Request is a type for sending
// form values and form files to
// the Gotenberg API.
type Request interface {
	SetWebhookURL(webhookURL string)
	getPostURL() string
	getFormValues() map[string]string
	getFormFiles() map[string]string
}

// ChromeRequest is a type for sending
// conversion requests which will be
// handle by Google Chrome.
type ChromeRequest interface {
	SetHeader(fpath string) error
	SetFooter(fpath string) error
	SetPaperSize(size [2]float64)
	SetMargins(margins [4]float64)
	SetLandscape(isLandscape bool)
	SetWebFontsTimeout(timeout int64)
}

// UnoconvRequest is a type for sending
// conversion requests which will be
// handle by unoconv.
type UnoconvRequest interface {
	SetLandscape(landscape bool)
}

// Post sends a request to the Gotenberg API
// and returns the response.
func (c *Client) Post(req Request) (*http.Response, error) {
	body, contentType, err := multipartForm(req)
	if err != nil {
		return nil, err
	}
	URL := fmt.Sprintf("%s%s", c.Hostname, req.getPostURL())
	resp, err := http.Post(URL, contentType, body)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Store creates the resulting PDF to given destination.
func (c *Client) Store(req Request, dest string) error {
	if hasWebhook(req) {
		return errors.New("cannot use Store method with a webhook")
	}
	resp, err := c.Post(req)
	if err != nil {
		return err
	}
	return writeNewFile(dest, resp.Body)
}

func hasWebhook(req Request) bool {
	webhookURL, ok := req.getFormValues()[webhookURL]
	if !ok {
		return false
	}
	return webhookURL != ""
}

func writeNewFile(fpath string, in io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
		return fmt.Errorf("%s: making directory for file: %v", fpath, err)
	}
	out, err := os.Create(fpath)
	if err != nil {
		return fmt.Errorf("%s: creating new file: %v", fpath, err)
	}
	defer out.Close()
	err = out.Chmod(0644)
	if err != nil && runtime.GOOS != "windows" {
		return fmt.Errorf("%s: changing file mode: %v", fpath, err)
	}
	_, err = io.Copy(out, in)
	if err != nil {
		return fmt.Errorf("%s: writing file: %v", fpath, err)
	}
	return nil
}

func fileExists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}

func multipartForm(req Request) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	defer writer.Close()
	for filename, fpath := range req.getFormFiles() {
		in, err := os.Open(fpath)
		if err != nil {
			return nil, "", fmt.Errorf("%s: opening file: %v", filename, err)
		}
		part, err := writer.CreateFormFile("files", filename)
		if err != nil {
			return nil, "", fmt.Errorf("%s: creating form file: %v", filename, err)
		}
		_, err = io.Copy(part, in)
		if err != nil {
			return nil, "", fmt.Errorf("%s: copying file: %v", filename, err)
		}
	}
	for name, value := range req.getFormValues() {
		if err := writer.WriteField(name, value); err != nil {
			return nil, "", fmt.Errorf("%s: writing form field: %v", name, err)
		}
	}
	return body, writer.FormDataContentType(), nil
}
