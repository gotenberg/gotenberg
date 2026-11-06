package pdf

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gotenberg/litho/pkg/models"
	"go.opentelemetry.io/otel/attribute"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

// extractedFile is an attachment written to disk, and the name it comes back
// under.
type extractedFile struct {
	path string
	name string
}

// extractAttachments writes the files each document embeds, the one s names
// or every one, and returns their paths, document after document. A file
// factur-x.xml embedded in invoice.pdf comes back as invoice_factur-x.xml, so
// that documents embedding the same name do not collide in the ZIP.
func (p *Processor) extractAttachments(ctx *api.Context, s *extractAttachmentsStage, docs []document, password string) ([]string, error) {
	extracted := make([][]extractedFile, len(docs))

	err := p.forEachInputPathIndexed(ctx, documentPaths(docs), func(i int, _ string) error {
		var err error
		extracted[i], err = p.extractAttachmentsOne(ctx, s, docs[i], password)
		if err != nil {
			return &stageError{op: s.op(), filename: docs[i].filename, err: err}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Registered here rather than per document: the request's filename map
	// is not safe for concurrent writes, and the names must be unique across
	// documents.
	var paths []string
	taken := make(map[string]bool)

	for _, files := range extracted {
		for _, f := range files {
			ctx.RegisterDiskPath(f.path, uniqueName(f.name, taken))
			paths = append(paths, f.path)
		}
	}

	return paths, nil
}

func (p *Processor) extractAttachmentsOne(ctx *api.Context, s *extractAttachmentsStage, d document, password string) (_ []extractedFile, err error) {
	spanCtx, span := gotenberg.Tracer().Start(ctx, "pdf.extract-attachments")
	defer func() {
		_ = gotenberg.SetSpanStatus(span, err, errorType(err))
		span.End()
	}()

	r, err := p.open(spanCtx, d.path, password)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()

	embedded, err := r.EmbeddedFiles(spanCtx)
	if err != nil {
		return nil, fmt.Errorf("read the embedded files: %w", err)
	}

	files, err := selectAttachments(embedded, s.Name)
	if err != nil {
		return nil, err
	}

	span.SetAttributes(attribute.Int("gotenberg.pdf.attachment.count", len(files)))

	base := strings.TrimSuffix(d.filename, filepath.Ext(d.filename))
	out := make([]extractedFile, 0, len(files))

	for _, f := range files {
		name := attachmentFilename(base, f.Name)
		path := ctx.GeneratePath(filepath.Ext(name))

		err = writeAttachment(spanCtx, path, f)
		if err != nil {
			return nil, fmt.Errorf("write attachment %q: %w", f.Name, err)
		}

		out = append(out, extractedFile{path: path, name: name})
	}

	return out, nil
}

// selectAttachments returns the files named name, or every file when name is
// empty. Finding none is the client's mistake, not an empty answer.
func selectAttachments(files []models.EmbeddedFile, name string) ([]models.EmbeddedFile, error) {
	if name == "" {
		if len(files) == 0 {
			return nil, &stageInputError{message: "the PDF has no attachments"}
		}

		return files, nil
	}

	var selected []models.EmbeddedFile
	var names []string

	for _, f := range files {
		if f.Name == name {
			selected = append(selected, f)
		}
		names = append(names, f.Name)
	}

	if len(selected) == 0 {
		return nil, missingAttachmentError(name, names)
	}

	return selected, nil
}

// attachmentFilename returns the name an attachment comes back under: the
// document's base name, then the attachment's. The attachment's name comes
// from the PDF, so it loses any directory and control character, which would
// otherwise reach the ZIP as a path.
func attachmentFilename(base, name string) string {
	name = api.SanitizeFilename(name)
	if name == "" || name == "." || name == ".." {
		// litho names a file specification without a name the same way.
		name = "embedded"
	}

	return base + "_" + name
}

// uniqueName returns name, or a numbered variant of it, invoice_factur-x_2.xml
// for instance, when taken already holds it, and records the result in taken.
func uniqueName(name string, taken map[string]bool) string {
	candidate := name
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)

	for i := 2; taken[candidate]; i++ {
		candidate = fmt.Sprintf("%s_%d%s", stem, i, ext)
	}

	taken[candidate] = true

	return candidate
}

// writeAttachment streams the decoded content of f into a new file at path.
func writeAttachment(ctx context.Context, path string, f models.EmbeddedFile) error {
	if f.Open == nil {
		return fmt.Errorf("attachment %q has no content", f.Name)
	}

	in, err := f.Open(ctx)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(path) //nolint:gosec // path is generated within the request's directory.
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	_, err = io.Copy(out, in)
	closeErr := out.Close()

	if err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	if closeErr != nil {
		return fmt.Errorf("close file: %w", closeErr)
	}

	return nil
}
