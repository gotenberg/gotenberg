// Package handlers implements all functions on which a request will pass through.
package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gulien/gotenberg/app/handlers/context"
	"github.com/gulien/gotenberg/app/handlers/converter"
	ghttp "github.com/gulien/gotenberg/app/handlers/http"
	"github.com/gulien/gotenberg/app/logger"

	"github.com/justinas/alice"
)

// GetHandlersChain returns the handlers chaining
// thanks to the alice library.
func GetHandlersChain() http.Handler {
	return alice.New(enforceContentLengthHandler, enforceContentTypeHandler, convertHandler, serveHandler).ThenFunc(clearHandler)
}

// requestHasNoContentError is raised when the request
// content length is 0.
type requestHasNoContentError struct{}

func (e *requestHasNoContentError) Error() string {
	return "Request has not content"
}

// enforeContentLengthHandler checks if the request has content.
func enforceContentLengthHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength == 0 {
			e := &requestHasNoContentError{}
			http.Error(w, e.Error(), http.StatusBadRequest)
			logger.Error(e)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// enforceContentTypeHandler checks if the "Content-Type" entry
// from the request's header matches one of the allowed content types.
func enforceContentTypeHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct, err := ghttp.FindAuthorizedContentType(r.Header)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
			logger.Error(err)
			return
		}

		r = context.WithContentType(r, ct)

		next.ServeHTTP(w, r)
	})
}

// convertHandler is in charge of converting the file(s) from the request to PDF.
func convertHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct, err := context.GetContentType(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logger.Error(err)
			return
		}

		c, err := converter.NewConverter(r, ct)
		if err != nil {
			if noFileToConvertError, ok := err.(*converter.NoFileToConvertError); ok {
				http.Error(w, noFileToConvertError.Error(), http.StatusBadRequest)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}

			logger.Error(err)
			return
		}

		path, err := c.Convert()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logger.Error(err)
			return
		}

		r = context.WithConverter(r, c)
		r = context.WithResultFilePath(r, path)

		next.ServeHTTP(w, r)
	})
}

// serveHandler simply serves the created PDF.
func serveHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, err := context.GetResultFilePath(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logger.Error(err)
		}

		reader, err := os.Open(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logger.Error(err)
			return
		}

		defer reader.Close()

		resultFileInfo, err := reader.Stat()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logger.Error(err)
			return
		}

		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", resultFileInfo.Name()))
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", resultFileInfo.Size()))
		io.Copy(w, reader)

		next.ServeHTTP(w, r)
	})
}

// clearHandler removes all files created during the conversion.
func clearHandler(w http.ResponseWriter, r *http.Request) {
	c, err := context.GetConverter(r)
	if err != nil {
		logger.Warn(err.Error())
	}

	if err := c.Clear(); err != nil {
		logger.Warn(err.Error())
	}
}
