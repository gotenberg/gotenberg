package pdf

import (
	"bufio"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	litho "github.com/gotenberg/litho/pkg/pdf"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

// render draws the pages s selects of each document as images, and returns
// their paths, document after document, page after page. A document named
// invoice.pdf gives invoice_1.png, invoice_2.png, and so on, numbered by page.
func (p *Processor) render(ctx *api.Context, s *renderStage, docs []document, password string) ([]string, error) {
	images := make([][]string, len(docs))

	err := p.forEachInputPathIndexed(ctx, documentPaths(docs), func(i int, _ string) error {
		var err error
		images[i], err = p.renderOne(ctx, s, docs[i], password)
		if err != nil {
			return &stageError{op: s.op(), filename: docs[i].filename, err: err}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	var out []string
	for _, paths := range images {
		out = append(out, paths...)
	}

	return out, nil
}

func (p *Processor) renderOne(ctx *api.Context, s *renderStage, d document, password string) (_ []string, err error) {
	spanCtx, span := gotenberg.Tracer().Start(ctx, "pdf.render",
		trace.WithAttributes(attribute.String("gotenberg.pdf.render.format", s.format())),
	)
	defer func() {
		_ = gotenberg.SetSpanStatus(span, err, errorType(err))
		span.End()
	}()

	r, err := p.open(spanCtx, d.path, password)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()

	pageCount, err := r.PageCount(spanCtx)
	if err != nil {
		return nil, fmt.Errorf("count pages: %w", err)
	}

	pages, err := renderPages(s.Pages, pageCount)
	if err != nil {
		return nil, err
	}

	span.SetAttributes(attribute.Int("gotenberg.pdf.render.pages", len(pages)))

	opts := []litho.RenderOption{litho.WithRenderDPI(float64(s.dpi()))}
	base := strings.TrimSuffix(d.filename, filepath.Ext(d.filename))
	paths := make([]string, 0, len(pages))

	// One page at a time: a page's raster is the memory a render holds.
	for _, page := range pages {
		img, err := r.RenderPage(spanCtx, page, opts...)
		if err != nil {
			return nil, fmt.Errorf("render page %d: %w", page, err)
		}

		path := ctx.GeneratePath("." + s.format())

		err = writeImage(path, img, s)
		if err != nil {
			return nil, fmt.Errorf("write page %d: %w", page, err)
		}

		ctx.RegisterDiskPath(path, fmt.Sprintf("%s_%d.%s", base, page, s.format()))
		paths = append(paths, path)
	}

	return paths, nil
}

// renderPages resolves the pages to draw: spec, or every page when it is
// empty.
func renderPages(spec string, pageCount int) ([]int, error) {
	if strings.TrimSpace(spec) != "" {
		return litho.ParsePageRanges(spec, pageCount)
	}

	pages := make([]int, pageCount)
	for i := range pages {
		pages[i] = i + 1
	}

	return pages, nil
}

// writeImage encodes img into a new file at path, in the format s selects.
func writeImage(path string, img image.Image, s *renderStage) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create image file: %w", err)
	}

	w := bufio.NewWriter(f)

	if s.format() == "jpeg" {
		err = jpeg.Encode(w, img, &jpeg.Options{Quality: s.quality()})
	} else {
		err = png.Encode(w, img)
	}

	if err == nil {
		err = w.Flush()
	}

	closeErr := f.Close()
	if err != nil {
		return fmt.Errorf("encode image: %w", err)
	}

	if closeErr != nil {
		return fmt.Errorf("close image file: %w", closeErr)
	}

	return nil
}

// dpi returns the resolution, litho's default when unset.
func (s *renderStage) dpi() int {
	if s.Dpi == 0 {
		return litho.DefaultRenderDPI
	}

	return s.Dpi
}

// format returns the image format, png when unset.
func (s *renderStage) format() string {
	if s.Format == "" {
		return "png"
	}

	return s.Format
}

// quality returns the jpeg quality, 100 when unset.
func (s *renderStage) quality() int {
	if s.Quality == 0 {
		return 100
	}

	return s.Quality
}
