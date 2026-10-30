package pdf

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gotenberg/litho/pkg/models"
	litho "github.com/gotenberg/litho/pkg/pdf"
	"github.com/labstack/echo/v5"

	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

// pipeRoute returns an [api.Route] which runs the stages of the "stages" form
// field on the uploaded PDFs.
func pipeRoute(processor *Processor) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/pdf/pipe",
		IsMultipart: true,
		Handler: func(c *echo.Context) error {
			ctx := c.Get("context").(*api.Context)

			form := ctx.FormData()

			stages, files, err := FormDataStages(form)
			if err != nil {
				return err
			}

			if len(stages) == 0 {
				return api.NewSentinelHttpError(
					http.StatusBadRequest,
					"Invalid form data: form field 'stages' is required, for example [{\"op\":\"merge\"}]",
				)
			}

			return runStages(ctx, form, processor, stages, files)
		},
	}
}

// stageRoute returns an [api.Route] which runs the single stage op, its
// options read from flat form fields.
func stageRoute(processor *Processor, op string) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/pdf/" + op,
		IsMultipart: true,
		Handler: func(c *echo.Context) error {
			ctx := c.Get("context").(*api.Context)

			form := ctx.FormData()

			stages, files, err := stageFromForm(op, form)
			if err != nil {
				return err
			}

			return runStages(ctx, form, processor, stages, files)
		},
	}
}

// runStages runs stages on the uploaded PDFs no stage reads and registers the
// outputs. form must be the one the stages were read from, so that its
// validation reports a missing stage file.
func runStages(ctx *api.Context, form *api.FormData, processor *Processor, stages Stages, files map[string]string) error {
	inputPaths, err := inputPDFs(form, files)
	if err != nil {
		return err
	}

	outputPaths, err := processor.Run(ctx, stages, inputPaths, files)
	if err != nil {
		return fmt.Errorf("run stages: %w", err)
	}

	err = ctx.AddOutputPaths(outputPaths...)
	if err != nil {
		return fmt.Errorf("add output paths: %w", err)
	}

	return nil
}

// readRoute returns the [api.Route] /forms/pdf/{name}, which answers, for
// every uploaded PDF, the JSON value read returns, keyed by filename. A
// "password" form field opens encrypted PDFs.
func readRoute(processor *Processor, name string, read func(ctx context.Context, r *litho.Reader, form *api.FormData) (any, error)) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/pdf/" + name,
		IsMultipart: true,
		Handler: func(c *echo.Context) error {
			ctx := c.Get("context").(*api.Context)
			form := ctx.FormData()

			var inputPaths []string
			err := form.MandatoryPaths([]string{".pdf"}, &inputPaths).Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			var password string
			err = form.String("password", &password).Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			// Collected per index, then folded into the map on this
			// goroutine: a shared map cannot be written concurrently.
			collected := make([]any, len(inputPaths))
			err = processor.forEachInputPathIndexed(ctx, inputPaths, func(i int, inputPath string) error {
				filename := ctx.OriginalFilename(inputPath)

				r, err := processor.open(ctx, inputPath, password)
				if err != nil {
					return &stageError{op: name, filename: filename, err: err}
				}
				defer func() { _ = r.Close() }()

				collected[i], err = read(ctx, r, form)
				if err != nil {
					return &stageError{op: name, filename: filename, err: err}
				}

				return nil
			})
			if err != nil {
				return fmt.Errorf("read PDFs: %w", handleError(err, ctx.DirPath()))
			}

			res := make(map[string]any, len(inputPaths))
			for i, inputPath := range inputPaths {
				res[ctx.OriginalFilename(inputPath)] = collected[i]
			}

			err = c.JSON(http.StatusOK, res)
			if err != nil {
				return fmt.Errorf("return JSON response: %w", err)
			}

			return api.ErrNoOutputFile
		},
	}
}

func metadataRoute(processor *Processor) api.Route {
	return readRoute(processor, "metadata", func(ctx context.Context, r *litho.Reader, _ *api.FormData) (any, error) {
		return r.Metadata(ctx)
	})
}

func bookmarksRoute(processor *Processor) api.Route {
	return readRoute(processor, "bookmarks", func(ctx context.Context, r *litho.Reader, _ *api.FormData) (any, error) {
		bookmarks, err := r.Bookmarks(ctx)
		if bookmarks == nil {
			bookmarks = []models.Bookmark{}
		}

		return bookmarks, err
	})
}

// documentInfo is the JSON answer of the info route.
type documentInfo struct {
	Pages      int    `json:"pages"`
	Version    string `json:"version"`
	Encrypted  bool   `json:"encrypted"`
	Tagged     bool   `json:"tagged"`
	Linearized bool   `json:"linearized"`
	FormFields bool   `json:"formFields"`
}

func infoRoute(processor *Processor) api.Route {
	return readRoute(processor, "info", func(ctx context.Context, r *litho.Reader, _ *api.FormData) (any, error) {
		pages, err := r.PageCount(ctx)
		if err != nil {
			return nil, fmt.Errorf("count pages: %w", err)
		}

		tagged, err := r.IsTagged(ctx)
		if err != nil {
			return nil, fmt.Errorf("read tagging: %w", err)
		}

		formFields, err := r.HasFormFields(ctx)
		if err != nil {
			return nil, fmt.Errorf("read form fields: %w", err)
		}

		return documentInfo{
			Pages:      pages,
			Version:    r.Version(),
			Encrypted:  r.IsEncrypted(),
			Tagged:     tagged,
			Linearized: r.IsLinearized(),
			FormFields: formFields,
		}, nil
	})
}

// violation is one conformance failure in the validate route's answer.
type violation struct {
	Specification string `json:"specification,omitempty"`
	Clause        string `json:"clause,omitempty"`
	TestNumber    int    `json:"testNumber,omitempty"`
	Description   string `json:"description"`
	Context       string `json:"context,omitempty"`
	Severity      string `json:"severity,omitempty"`
}

// validation is the JSON answer of the validate route.
type validation struct {
	Profile    string      `json:"profile"`
	Compliant  bool        `json:"compliant"`
	Violations []violation `json:"violations"`
}

func validateRoute(processor *Processor) api.Route {
	return readRoute(processor, "validate", func(ctx context.Context, r *litho.Reader, form *api.FormData) (any, error) {
		var profile string
		err := form.MandatoryString("profile", &profile).Validate()
		if err != nil {
			return nil, fmt.Errorf("validate form data: %w", err)
		}

		var found []litho.Violation
		switch {
		case pdfaProfiles[profile] != "":
			found, err = r.ValidatePDFAReport(ctx, pdfaProfiles[profile])
		case pdfuaProfiles[profile] != "":
			found, err = r.ValidatePDFUAReport(ctx, pdfuaProfiles[profile])
		case profile == facturXProfile:
			var report litho.FacturXReport
			report, err = r.ValidateFacturXReport(ctx)
			for _, v := range report.Violations {
				found = append(found, litho.Violation{Specification: facturXProfile, Description: v.Error()})
			}
		default:
			return nil, api.NewSentinelHttpError(
				http.StatusBadRequest,
				fmt.Sprintf("Invalid form data: form field 'profile' must be one of %s, PDF/UA-1, PDF/UA-2 or %s", strings.Join(sortedKeys(pdfaProfiles), ", "), facturXProfile),
			)
		}

		if err != nil {
			return nil, fmt.Errorf("validate against %s: %w", profile, err)
		}

		res := validation{Profile: profile, Compliant: len(found) == 0, Violations: make([]violation, len(found))}
		for i, v := range found {
			res.Violations[i] = violation{
				Specification: v.Specification,
				Clause:        v.Clause,
				TestNumber:    v.TestNumber,
				Description:   v.Description,
				Context:       v.Context,
				Severity:      v.Severity.String(),
			}
		}

		return res, nil
	})
}

func attachmentsRoute(processor *Processor) api.Route {
	return readRoute(processor, "attachments", func(ctx context.Context, r *litho.Reader, _ *api.FormData) (any, error) {
		files, err := r.EmbeddedFiles(ctx)
		if files == nil {
			files = []models.EmbeddedFile{}
		}

		return files, err
	})
}
