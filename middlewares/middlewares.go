/*
Package middlewares implements all handlers of the application.

Those handlers manage requests coming from the unique entry point (aka "/").
*/
package middlewares

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gulien/gotenberg/context"
	"github.com/gulien/gotenberg/converters"
	"github.com/gulien/gotenberg/helpers"
	"github.com/gulien/gotenberg/logger"

	"github.com/justinas/alice"
	"github.com/satori/go.uuid"
)

// GetMiddlewaresChain builds and returns the chaining of handlers
// using the alice library.
func GetMiddlewaresChain() http.Handler {
	return alice.New(loggingHandler, enforceContentLengthHandler, enforceContentTypeHandler, convertHandler, serveHandler).ThenFunc(clearHandler)
}

// loggingHandler identifies the request.
func loggingHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		transactionID := uuid.NewV4().String()
		r = r.WithContext(context.WithTransactionID(r.Context(), transactionID))
		logger.InfoR(context.GetTransactionID(r.Context()), fmt.Sprintf("Hello %s", r.RemoteAddr))
		next.ServeHTTP(w, r)
	})
}

// enforeContentLengthHandler checks if the request has content.
func enforceContentLengthHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength == 0 {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			logger.ErrorR(context.GetTransactionID(r.Context()), fmt.Errorf("%s", http.StatusText(http.StatusBadRequest)), http.StatusBadRequest, "Request has no content")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// enforceContentTypeHandler checks if the "Content-Type" entry
// from the request's header matches one of the allowed content types.
func enforceContentTypeHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := helpers.GetMatchingContentType(r.Header.Get("Content-Type"))
		if contentType == "" {
			http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
			logger.ErrorR(context.GetTransactionID(r.Context()), fmt.Errorf("%s", http.StatusText(http.StatusUnsupportedMediaType)), http.StatusUnsupportedMediaType, "No matching content type found")
			return
		}

		r = r.WithContext(context.WithContentType(r.Context(), contentType))
		next.ServeHTTP(w, r)
	})
}

// convertHandler is in charge of converting the file(s) from the request to PDF.
func convertHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := context.GetContentType(r.Context())

		c, err := converters.NewConverter(contentType, r)
		if err != nil {
			if converterUnprocessableEntityError, ok := err.(*converters.ConverterUnprocessableEntityError); ok {
				// a file from the request has a content type which does not match with one of the allowed
				// content types.
				http.Error(w, converterUnprocessableEntityError.Error(), http.StatusUnprocessableEntity)
				logger.ErrorR(context.GetTransactionID(r.Context()), converterUnprocessableEntityError, http.StatusUnprocessableEntity, "A file has a content type which does not match with one of the allowed content types")
			} else {
				// something bad happened during the creation of the converter...
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				logger.ErrorR(context.GetTransactionID(r.Context()), err, http.StatusInternalServerError, "Something bad happened during the creation of the converter")
			}

			return
		}

		resultFilePath, err := c.Convert()
		if err != nil {
			// an error occured during conversion...
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			logger.ErrorR(context.GetTransactionID(r.Context()), err, http.StatusInternalServerError, "An error occured during conversion")
			return
		}

		r = r.WithContext(context.WithResultFilePath(r.Context(), resultFilePath))
		r = r.WithContext(context.WithConverter(r.Context(), c))
		next.ServeHTTP(w, r)
	})
}

// serveHandler simply serves the created PDF.
func serveHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resultFilePath := context.GetResultFilePath(r.Context())

		reader, err := os.Open(resultFilePath)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			logger.ErrorR(context.GetTransactionID(r.Context()), err, http.StatusInternalServerError, fmt.Sprintf("An error occured while opening result file \"%s\"", resultFilePath))
			return
		}

		defer reader.Close()

		resultFileInfo, err := reader.Stat()
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			logger.ErrorR(context.GetTransactionID(r.Context()), err, http.StatusInternalServerError, fmt.Sprintf("An error occured while retrieving info from result file \"%s\"", resultFilePath))
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
	c := context.GetConverter(r.Context())

	if err := c.Clear(); err != nil {
		logger.WarnR(context.GetTransactionID(r.Context()), err.Error())
	}
	logger.InfoR(context.GetTransactionID(r.Context()), fmt.Sprintf("Bye %s", r.RemoteAddr))
}
