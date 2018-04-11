// Package app implements all functions on which a request will pass through.
package app

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/thecodingmachine/gotenberg/app/context"
	"github.com/thecodingmachine/gotenberg/app/converter"
	ghttp "github.com/thecodingmachine/gotenberg/app/http"
	"github.com/thecodingmachine/gotenberg/app/logger"

	"github.com/justinas/alice"
)

// GetHandlersChain returns the handlers chaining
// thanks to the alice library.
func GetHandlersChain() http.Handler {
	return alice.New(enforceContentLengthHandler, enforceContentTypeHandler, convertHandler).ThenFunc(serveHandler)
}

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
// from the request's header matches the allowed content type.
func enforceContentTypeHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := ghttp.CheckAuthorizedContentType(r.Header); err != nil {
			http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
			logger.Error(err)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// convertHandler is in charge of converting the file(s) from the request to PDF.
func convertHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := converter.NewConverter(r)
		if err != nil {
			// TODO bad request if no file to convert
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logger.Error(err)

			if c != nil {
				r = context.WithConverter(r, c)
				cleanup(r)
			}

			return
		}

		r = context.WithConverter(r, c)

		path, err := c.Convert()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logger.Error(err)
			cleanup(r)
			return
		}

		r = context.WithResultFilePath(r, path)

		next.ServeHTTP(w, r)
	})
}

// serveHandler simply serves the created PDF.
func serveHandler(w http.ResponseWriter, r *http.Request) {
	path, err := context.GetResultFilePath(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.Error(err)
		cleanup(r)
		return
	}

	reader, err := os.Open(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.Error(err)
		cleanup(r)
		return
	}

	defer reader.Close()

	resultFileInfo, err := reader.Stat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.Error(err)
		cleanup(r)
		return
	}

	done := make(chan error, 1)
	go func() {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", resultFileInfo.Name()))
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", resultFileInfo.Size()))
		_, err := io.Copy(w, reader)

		done <- err
	}()

	err = <-done
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.Error(err)
	}

	cleanup(r)
}

// cleanup removes all files created during the conversion.
func cleanup(r *http.Request) {
	c, err := context.GetConverter(r)
	if err != nil {
		logger.Warn(err.Error())
		return
	}

	if err := c.Clear(); err != nil {
		logger.Warn(err.Error())
	}
}
