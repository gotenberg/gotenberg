package pdf

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	litho "github.com/gotenberg/litho/pkg/pdf"
	"go.opentelemetry.io/otel/attribute"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

// pageSeparator joins the text of the pages, as pdftotext does.
const pageSeparator = "\f"

// extractText writes the text of the pages s selects of each document, and
// returns the paths of the .txt files, one per document: invoice.pdf gives
// invoice.txt.
func (p *Processor) extractText(ctx *api.Context, s *extractTextStage, docs []document, password string) ([]string, error) {
	extracted := make([]extractedFile, len(docs))

	err := p.forEachInputPathIndexed(ctx, documentPaths(docs), func(i int, _ string) error {
		var err error
		extracted[i], err = p.extractTextOne(ctx, s, docs[i], password)
		if err != nil {
			return &stageError{op: s.op(), filename: docs[i].filename, err: err}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Named here rather than per document: invoice.pdf and invoice.PDF would
	// both give invoice.txt, and the numbering follows the document order.
	paths := make([]string, len(extracted))
	taken := make(map[string]bool)

	for i, f := range extracted {
		ctx.RegisterDiskPath(f.path, uniqueName(f.name, taken))
		paths[i] = f.path
	}

	return paths, nil
}

func (p *Processor) extractTextOne(ctx *api.Context, s *extractTextStage, d document, password string) (_ extractedFile, err error) {
	spanCtx, span := gotenberg.Tracer().Start(ctx, "pdf.extract-text")
	defer func() {
		_ = gotenberg.SetSpanStatus(span, err, errorType(err))
		span.End()
	}()

	r, err := p.open(spanCtx, d.path, password)
	if err != nil {
		return extractedFile{}, err
	}
	defer func() { _ = r.Close() }()

	pageCount, err := r.PageCount(spanCtx)
	if err != nil {
		return extractedFile{}, fmt.Errorf("count pages: %w", err)
	}

	pages, err := selectedPages(s.Pages, pageCount)
	if err != nil {
		return extractedFile{}, err
	}

	span.SetAttributes(attribute.Int("gotenberg.pdf.page.count", len(pages)))

	opts := s.options()
	texts := make([]string, len(pages))

	for i, page := range pages {
		texts[i], err = r.ExtractText(spanCtx, page, opts...)
		// A scanned page holds an image, not text, and Gotenberg recognizes
		// none: it reads as empty rather than failing the document.
		if errors.Is(err, litho.ErrRequiresOCR) {
			texts[i], err = "", nil
		}
		if err != nil {
			return extractedFile{}, inputError(fmt.Sprintf("extract the text of page %d", page), err)
		}
	}

	path := ctx.GeneratePath(".txt")

	err = os.WriteFile(path, []byte(strings.Join(texts, pageSeparator)), 0o600)
	if err != nil {
		return extractedFile{}, fmt.Errorf("write text: %w", err)
	}

	base := strings.TrimSuffix(d.filename, filepath.Ext(d.filename))

	return extractedFile{path: path, name: base + ".txt"}, nil
}

// options returns the litho options of the text s reads besides what the page
// shows.
func (s *extractTextStage) options() []litho.ExtractTextOption {
	var opts []litho.ExtractTextOption
	if s.Invisible {
		opts = append(opts, litho.WithInvisibleText())
	}
	if s.Artifacts {
		opts = append(opts, litho.WithArtifacts())
	}
	if s.Annotations {
		opts = append(opts, litho.WithAnnotationText())
	}

	return opts
}
