package pdf

import (
	"context"
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
// returns the paths of the files, one per document: invoice.pdf gives
// invoice.txt, or invoice.md for Markdown.
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

	span.SetAttributes(
		attribute.Int("gotenberg.pdf.page.count", len(pages)),
		attribute.String("gotenberg.pdf.text.format", s.formatName()),
	)

	var text, ext string
	if s.markdown() {
		ext = ".md"
		text, err = r.ExtractMarkdown(spanCtx, pages, s.options()...)
		if err != nil {
			return extractedFile{}, inputError("extract the text as Markdown", err)
		}
	} else {
		ext = ".txt"
		text, err = plainText(spanCtx, r, pages, s.options())
		if err != nil {
			return extractedFile{}, err
		}
	}

	path := ctx.GeneratePath(ext)

	err = os.WriteFile(path, []byte(text), 0o600)
	if err != nil {
		return extractedFile{}, fmt.Errorf("write text: %w", err)
	}

	base := strings.TrimSuffix(d.filename, filepath.Ext(d.filename))

	return extractedFile{path: path, name: base + ext}, nil
}

// plainText returns the text of pages, in the order given, joined by
// [pageSeparator].
func plainText(ctx context.Context, r *litho.Reader, pages []int, opts []litho.ExtractTextOption) (string, error) {
	texts := make([]string, len(pages))

	for i, page := range pages {
		var err error
		texts[i], err = r.ExtractText(ctx, page, opts...)
		// A scanned page holds an image, not text, and Gotenberg recognizes
		// none: it reads as empty rather than failing the document.
		if errors.Is(err, litho.ErrRequiresOCR) {
			texts[i], err = "", nil
		}
		if err != nil {
			return "", inputError(fmt.Sprintf("extract the text of page %d", page), err)
		}
	}

	return strings.Join(texts, pageSeparator), nil
}

// formatName returns the format s writes, its default named.
func (s *extractTextStage) formatName() string {
	if s.Format == "" {
		return textFormatText
	}

	return s.Format
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
