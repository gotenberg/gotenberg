package pdf

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
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

	// A render alone leaves the uploaded PDFs as they are: answer the images
	// only.
	if len(stages) == 1 && stages[0].op() == "render" {
		outputPaths = slices.DeleteFunc(outputPaths, func(path string) bool {
			return strings.EqualFold(filepath.Ext(path), ".pdf")
		})
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
//
// newRead binds the route's own form fields, which readRoute validates with
// the others before it opens a PDF, and returns the read of one PDF. It runs
// once per request.
func readRoute(processor *Processor, name string, newRead func(form *api.FormData) readFunc) api.Route {
	return api.Route{
		Method:          http.MethodPost,
		Path:            "/forms/pdf/" + name,
		IsMultipart:     true,
		AnswersDirectly: true,
		Handler: func(c *echo.Context) error {
			ctx := c.Get("context").(*api.Context)
			form := ctx.FormData()

			var inputPaths []string
			err := form.MandatoryPaths([]string{".pdf"}, &inputPaths).Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			var password string
			form.String("password", &password)
			read := newRead(form)
			err = form.RejectUnknown(FormFieldHint).Validate()
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

				collected[i], err = read(ctx, r)
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

// readFunc reads one PDF for a read route, and returns its JSON answer.
type readFunc func(ctx context.Context, r *litho.Reader) (any, error)

// noFields is the newRead of a read route that takes no field of its own.
func noFields(read readFunc) func(form *api.FormData) readFunc {
	return func(*api.FormData) readFunc { return read }
}

// documentMetadata is the JSON answer of the metadata route. Its keys are the
// options of the set-metadata stage, so that an answer feeds the stage as it
// is. A field the document does not carry is left out.
type documentMetadata struct {
	Title        string `json:"title,omitempty"`
	Author       string `json:"author,omitempty"`
	Subject      string `json:"subject,omitempty"`
	Keywords     string `json:"keywords,omitempty"`
	Creator      string `json:"creator,omitempty"`
	Producer     string `json:"producer,omitempty"`
	CreationDate string `json:"creationDate,omitempty"`
	ModDate      string `json:"modDate,omitempty"`
}

func metadataRoute(processor *Processor) api.Route {
	return readRoute(processor, "metadata", noFields(func(ctx context.Context, r *litho.Reader) (any, error) {
		metadata, err := r.Metadata(ctx)
		// A document without an /Info dictionary or an XMP packet is valid,
		// and has no metadata to answer.
		if errors.Is(err, litho.ErrNoMetadata) {
			return documentMetadata{}, nil
		}

		if err != nil {
			return nil, fmt.Errorf("read metadata: %w", err)
		}

		// litho names the fields after the /Info keys (ISO 32000-1, Table
		// 317). It also reads the catalog's /Lang and the XMP rights, which
		// the set-metadata stage cannot write: they are left out.
		return documentMetadata{
			Title:        metadata["Title"],
			Author:       metadata["Author"],
			Subject:      metadata["Subject"],
			Keywords:     metadata["Keywords"],
			Creator:      metadata["Creator"],
			Producer:     metadata["Producer"],
			CreationDate: metadata["CreationDate"],
			ModDate:      metadata["ModDate"],
		}, nil
	}))
}

func bookmarksRoute(processor *Processor) api.Route {
	return readRoute(processor, "bookmarks", noFields(func(ctx context.Context, r *litho.Reader) (any, error) {
		bookmarks, err := r.Bookmarks(ctx)
		if bookmarks == nil {
			bookmarks = []models.Bookmark{}
		}

		return bookmarks, err
	}))
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
	return readRoute(processor, "info", noFields(func(ctx context.Context, r *litho.Reader) (any, error) {
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
	}))
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
	return readRoute(processor, "validate", func(form *api.FormData) readFunc {
		var profile string
		form.MandatoryString("profile", &profile)

		return func(ctx context.Context, r *litho.Reader) (any, error) {
			return validatePDF(ctx, r, profile)
		}
	})
}

// validatePDF validates the PDF r reads against profile.
func validatePDF(ctx context.Context, r *litho.Reader, profile string) (any, error) {
	var (
		found []litho.Violation
		err   error
	)

	switch {
	case pdfaProfiles[profile] != "":
		found, err = r.ValidatePDFAReport(ctx, pdfaProfiles[profile])
	case pdfuaProfiles[profile] != "":
		found, err = r.ValidatePDFUAReport(ctx, pdfuaProfiles[profile])
	case profile == facturXProfile:
		var report litho.FacturXReport
		report, err = r.ValidateFacturXReport(ctx)
		found = report.Structured()
	default:
		return nil, api.NewSentinelHttpError(
			http.StatusBadRequest,
			fmt.Sprintf("Invalid form data: form field 'profile' must be one of %s, PDF/UA-1, PDF/UA-2 or %s", strings.Join(sortedKeys(pdfaProfiles), ", "), facturXProfile),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("validate against %s: %w", profile, err)
	}

	// A Factur-X report holds warnings too, soft Schematron findings that
	// do not by themselves fail conformance.
	res := validation{
		Profile:    profile,
		Compliant:  !slices.ContainsFunc(found, func(v litho.Violation) bool { return v.Severity == litho.SeverityFatal }),
		Violations: make([]violation, len(found)),
	}
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
}

func attachmentsRoute(processor *Processor) api.Route {
	return readRoute(processor, "attachments", noFields(func(ctx context.Context, r *litho.Reader) (any, error) {
		files, err := r.EmbeddedFiles(ctx)
		if files == nil {
			files = []models.EmbeddedFile{}
		}

		return files, err
	}))
}
