package resource

import (
	"io"
	"io/ioutil"
	"os"

	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
)

// file represents a file within the resource.
type file struct {
	fpath string
}

// write writes given content to the
// resourceFile location.
func (f file) write(in io.Reader) error {
	const op string = "resource.file.write"
	resolver := func() error {
		out, err := os.Create(f.fpath)
		if err != nil {
			return err
		}
		defer out.Close() // nolint: errcheck
		if err := out.Chmod(0644); err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			return err
		}
		if _, err := out.Seek(0, 0); err != nil {
			return err
		}
		return nil
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

// content returns the string content of
// the file.
func (f file) content() (string, error) {
	const op string = "resource.file.content"
	b, err := ioutil.ReadFile(f.fpath)
	if err != nil {
		return "", xerror.New(op, err)
	}
	return string(b), nil
}

/*
HeaderFooterContents is a helper for retrieving
the content of the files "header.html"
and "footer.html".
*/
func HeaderFooterContents(r Resource) (string, string, error) {
	const (
		op                      string = "resource.HeaderFooterContents"
		defaultHeaderFooterHTML string = "<html><head></head><body></body></html>"
	)
	resolver := func() (string, string, error) {
		headerHTML, err := r.Fcontent("header.html", defaultHeaderFooterHTML)
		if err != nil {
			return defaultHeaderFooterHTML,
				defaultHeaderFooterHTML,
				err
		}
		footerHTML, err := r.Fcontent("footer.html", defaultHeaderFooterHTML)
		if err != nil {
			return defaultHeaderFooterHTML,
				defaultHeaderFooterHTML,
				err
		}
		return headerHTML,
			footerHTML,
			nil
	}
	headerHTML, footerHTML,
		err := resolver()
	if err != nil {
		return headerHTML,
			footerHTML,
			xerror.New(op, err)
	}
	return headerHTML,
		footerHTML,
		nil
}
