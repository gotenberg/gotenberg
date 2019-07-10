package handler

import (
	"fmt"
	"net/http"
	"os"

	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/context"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/resource"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
	"github.com/thecodingmachine/gotenberg/internal/pkg/random"
	"github.com/thecodingmachine/gotenberg/internal/pkg/standarderror"
)

const (
	// PingEndpoint is the route for healthcheck.
	PingEndpoint = "/ping"
	// MergeEndpoint is the route for merging PDF files.
	MergeEndpoint = "/merge"
	// ConvertGroupEndpoint is the route of the group
	// in charge of converting files to PDF.
	ConvertGroupEndpoint = "/convert"
	// HTMLEndpoint is the route for converting
	// HTML to PDF.
	HTMLEndpoint = "/html"
	// URLEndpoint is the route for converting
	// a URL to PDF.
	URLEndpoint = "/url"
	// MarkdownEndpoint is the route for converting
	// Markdown to PDF.
	MarkdownEndpoint = "/markdown"
	// OfficeEndpoint is the route for converting
	// Office files to PDF.
	OfficeEndpoint = "/office"
)

func convert(ctx *context.Context, p printer.Printer) error {
	const op string = "handler.convert"
	r := ctx.Resource()
	logger := ctx.StandardLogger()
	baseFilename := random.Get()
	filename := fmt.Sprintf("%s.pdf", baseFilename)
	fpath := fmt.Sprintf("%s/%s", r.DirPath(), filename)
	// if no webhook URL given, run conversion
	// and directly return the resulting PDF file
	// or an error.
	if !r.Has(resource.WebhookURLFormField) {
		logger.DebugfOp(op, "no '%s' found, converting synchronously", resource.WebhookURLFormField)
		if err := convertSync(filename, fpath, ctx, p); err != nil {
			return &standarderror.Error{Op: op, Err: err}
		}
		return nil
	}
	// as a webhook URL has been given, we
	// run the following lines in a goroutine so that
	// it doesn't block.
	logger.DebugfOp(op, "'%s' found, converting asynchronously", resource.WebhookURLFormField)
	return convertAsync(filename, fpath, ctx, p)
}

func convertSync(filename, fpath string, ctx *context.Context, p printer.Printer) error {
	const op = "handler.convertSync"
	r := ctx.Resource()
	logger := ctx.StandardLogger()
	if err := p.Print(fpath); err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	if !r.Has(resource.ResultFilenameFormField) {
		logger.DebugfOp(
			op,
			"no '%s' found, using generated filename '%s'",
			resource.ResultFilenameFormField,
			filename,
		)
		if err := ctx.Attachment(fpath, filename); err != nil {
			return &standarderror.Error{Op: op, Err: err}
		}
		return nil
	}
	logger.DebugfOp(
		op,
		"'%s' found, so not using generated filename",
		resource.ResultFilenameFormField,
	)
	filename, err := r.Get(resource.ResultFilenameFormField)
	if err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	if err := ctx.Attachment(fpath, filename); err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	return nil
}

func convertAsync(filename, fpath string, ctx *context.Context, p printer.Printer) error {
	const op = "handler.convertAsync"
	r := ctx.Resource()
	logger := ctx.StandardLogger()
	go func() {
		defer r.Close() // nolint: errcheck
		if err := p.Print(fpath); err != nil {
			logger.ErrorOp(
				op,
				&standarderror.Error{Op: op, Err: err},
			)
			return
		}
		f, err := os.Open(fpath)
		if err != nil {
			logger.ErrorOp(
				op,
				&standarderror.Error{Op: op, Err: err},
			)
			return
		}
		defer f.Close() // nolint: errcheck
		webhookURL, err := r.Get(resource.WebhookURLFormField)
		if err != nil {
			logger.ErrorOp(
				op,
				&standarderror.Error{Op: op, Err: err},
			)
			return
		}
		logger.DebugfOp(
			op,
			"sending result file '%s' to '%s'",
			filename,
			webhookURL,
		)
		resp, err := http.Post(webhookURL, "application/pdf", f) /* #nosec */
		if err != nil {
			logger.ErrorOp(
				op,
				&standarderror.Error{Op: op, Err: err},
			)
			return
		}
		defer resp.Body.Close() // nolint: errcheck
	}()
	return nil
}
