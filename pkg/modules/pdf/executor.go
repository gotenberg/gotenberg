package pdf

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	litho "github.com/gotenberg/litho/pkg/pdf"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

// Processor runs [Stages] on PDF documents with litho. It is safe for
// concurrent use. The zero value uses litho's default limits and HTTP client,
// and remediates without the system's fonts.
type Processor struct {
	limits     litho.Limits
	fonts      litho.RemediateOption
	httpClient func(ctx context.Context) *http.Client
	files      fileConcurrency
}

// stageError locates a failure to a stage and a document, for the client.
type stageError struct {
	op       string
	filename string
	err      error
}

func (e *stageError) Error() string {
	return fmt.Sprintf("stage %q on %q: %s", e.op, e.filename, e.err)
}

func (e *stageError) Unwrap() error { return e.err }

// document is a PDF the stages produced or received, and the filename its
// output carries.
type document struct {
	path     string
	filename string
}

// Run runs stages on the PDFs at inputPaths, in order, and returns the paths
// of the resulting PDFs. files maps the uploaded filenames the stages read to
// their paths on disk.
//
// A document keeps its filename through the stages. "merge" names its output
// after the first document, and "split" numbers its outputs. Without stages,
// Run returns inputPaths.
//
// A failure the request causes, such as a page range past the last page or a
// wrong password, returns a 400 error naming the stage and the document.
//
// A "remediate" stage sets the [ResidualHeader] and [RolledBackHeader] output
// headers, summed over the documents, and logs the details of each.
func (p *Processor) Run(ctx *api.Context, stages Stages, inputPaths []string, files map[string]string) ([]string, error) {
	outcome := new(remediationOutcome)

	paths, err := p.run(ctx, stages, inputPaths, files, outcome)
	if err != nil {
		return nil, handleError(err, ctx.DirPath())
	}

	if stages.remediates() {
		ctx.SetOutputHeader(ResidualHeader, strconv.Itoa(outcome.residual))
		ctx.SetOutputHeader(RolledBackHeader, strconv.Itoa(outcome.rolledBack))
	}

	return paths, nil
}

func (p *Processor) run(ctx *api.Context, stages Stages, inputPaths []string, files map[string]string, outcome *remediationOutcome) ([]string, error) {
	docs := make([]document, len(inputPaths))
	for i, path := range inputPaths {
		docs[i] = document{path: path, filename: ctx.OriginalFilename(path)}
	}

	password := ""
	if len(stages) > 0 {
		if d, ok := stages[0].(*decryptStage); ok {
			password = d.Password
		}
	}

	var err error
	for i := 0; i < len(stages); {
		switch s := stages[i].(type) {
		case *mergeStage:
			docs, err = p.merge(ctx, docs, password)
			i++
		case *splitStage:
			docs, err = p.split(ctx, s, docs, password)
			i++
		default:
			end := runEnd(stages, i)
			docs, err = p.runEach(ctx, stages[i:end], docs, password, files, outcome)
			i = end
		}

		if err != nil {
			return nil, err
		}

		// Only inputs may be encrypted: decrypt is the first stage, and it
		// removes the encryption from everything it writes.
		password = ""
	}

	paths := make([]string, len(docs))
	for i, d := range docs {
		paths[i] = d.path
	}

	return paths, nil
}

// runEnd returns the index after the run of stages starting at start. A run
// shares one litho pipeline, so it stops before a merge or a split, which
// reshape the list of documents, and after a stage that changes the page
// count, so that every page range in a run counts the same pages.
func runEnd(stages Stages, start int) int {
	end := start
	for end < len(stages) {
		switch stages[end].(type) {
		case *mergeStage, *splitStage:
			return end
		case *selectStage, *deleteStage, *insertStage:
			return end + 1
		}

		end++
	}

	return end
}

// open opens the PDF at path with the processor's limits.
func (p *Processor) open(ctx context.Context, path, password string) (*litho.Reader, error) {
	opts := []litho.OpenOption{litho.WithLimits(p.limits)}
	if password != "" {
		opts = append(opts, litho.WithPassword(password))
	}

	r, err := litho.OpenFile(ctx, path, opts...)
	if err != nil {
		return nil, fmt.Errorf("open PDF: %w", err)
	}

	return r, nil
}

// write runs pipeline, the stages ops, into a new file named filename and
// returns its document. The pipeline runs its stages as it writes, so the
// pdf.write span holds the spans of the edits.
func (p *Processor) write(ctx *api.Context, pipeline *litho.Pipeline, ops []string, filename string, opts ...litho.WriteOption) (_ document, err error) {
	spanCtx, span := gotenberg.Tracer().Start(ctx, "pdf.write",
		trace.WithAttributes(attribute.StringSlice("gotenberg.pdf.stages", ops)),
	)
	defer func() {
		_ = gotenberg.SetSpanStatus(span, err, errorType(err))
		span.End()
	}()

	path := ctx.GeneratePath(".pdf")

	out, err := os.Create(path)
	if err != nil {
		return document{}, fmt.Errorf("create output file: %w", err)
	}

	var client *http.Client
	if p.httpClient != nil {
		client = p.httpClient(ctx)
	}

	writeCtx := litho.ContextWithHTTPClient(spanCtx, client)
	_, err = pipeline.WriteTo(writeCtx, out, append(opts, litho.WithPipelineTempDir(ctx.DirPath()))...)

	closeErr := out.Close()
	if err != nil {
		return document{}, err
	}

	if closeErr != nil {
		return document{}, fmt.Errorf("close output file: %w", closeErr)
	}

	ctx.RegisterDiskPath(path, filename)

	return document{path: path, filename: filename}, nil
}

// merge appends every document, in order, into one named after the first.
func (p *Processor) merge(ctx *api.Context, docs []document, password string) ([]document, error) {
	if len(docs) < 2 {
		return docs, nil
	}

	readers := make([]*litho.Reader, 0, len(docs))
	defer func() {
		for _, r := range readers {
			_ = r.Close()
		}
	}()

	for _, d := range docs {
		r, err := p.open(ctx, d.path, password)
		if err != nil {
			return nil, &stageError{op: "merge", filename: d.filename, err: err}
		}

		readers = append(readers, r)
	}

	pipeline := litho.NewPipeline().Open(readers[0]).Merge(readers[1:])

	merged, err := p.write(ctx, pipeline, []string{"merge"}, docs[0].filename)
	if err != nil {
		return nil, &stageError{op: "merge", filename: docs[0].filename, err: err}
	}

	return []document{merged}, nil
}

// split turns each document into several, numbered from 0 after it.
func (p *Processor) split(ctx *api.Context, s *splitStage, docs []document, password string) ([]document, error) {
	parts := make([][]document, len(docs))

	err := p.forEachInputPathIndexed(ctx, documentPaths(docs), func(i int, _ string) error {
		var err error
		parts[i], err = p.splitOne(ctx, s, docs[i], password)
		if err != nil {
			return &stageError{op: "split", filename: docs[i].filename, err: err}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	var out []document
	for _, part := range parts {
		out = append(out, part...)
	}

	return out, nil
}

func (p *Processor) splitOne(ctx *api.Context, s *splitStage, d document, password string) ([]document, error) {
	r, err := p.open(ctx, d.path, password)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()

	pageCount, err := r.PageCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("count pages: %w", err)
	}

	groups, err := splitGroups(s, pageCount)
	if err != nil {
		return nil, err
	}

	base := strings.TrimSuffix(d.filename, filepath.Ext(d.filename))
	parts := make([]document, 0, len(groups))

	for i, pages := range groups {
		part, err := p.write(ctx, litho.NewPipeline().Open(r).Split(pages), []string{"split"}, fmt.Sprintf("%s_%d.pdf", base, i))
		if err != nil {
			return nil, err
		}

		parts = append(parts, part)
	}

	return parts, nil
}

// splitGroups returns the pages of each part a split produces.
func splitGroups(s *splitStage, pageCount int) ([][]int, error) {
	var groups [][]int

	if s.Every > 0 {
		for start := 1; start <= pageCount; start += s.Every {
			var pages []int
			for page := start; page < start+s.Every && page <= pageCount; page++ {
				pages = append(pages, page)
			}

			groups = append(groups, pages)
		}

		return groups, nil
	}

	for part := range strings.SplitSeq(s.Ranges, ",") {
		pages, err := litho.ParsePageRanges(part, pageCount)
		if err != nil {
			return nil, err
		}

		groups = append(groups, pages)
	}

	return groups, nil
}

// runEach runs a run of stages on every document.
func (p *Processor) runEach(ctx *api.Context, run Stages, docs []document, password string, files map[string]string, outcome *remediationOutcome) ([]document, error) {
	out := make([]document, len(docs))

	err := p.forEachInputPathIndexed(ctx, documentPaths(docs), func(i int, _ string) error {
		var err error
		out[i], err = p.runOne(ctx, run, docs[i], password, files, outcome)

		return err
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

// runOne runs a run of stages on one document through a single litho
// pipeline.
func (p *Processor) runOne(ctx *api.Context, run Stages, d document, password string, files map[string]string, outcome *remediationOutcome) (document, error) {
	r, err := p.open(ctx, d.path, password)
	if err != nil {
		return document{}, &stageError{op: run[0].op(), filename: d.filename, err: err}
	}
	defer func() { _ = r.Close() }()

	pageCount, err := r.PageCount(ctx)
	if err != nil {
		return document{}, &stageError{op: run[0].op(), filename: d.filename, err: fmt.Errorf("count pages: %w", err)}
	}

	env := &runEnv{processor: p, pageCount: pageCount, files: files}
	defer env.close()

	pipeline := litho.NewPipeline().Open(r)
	var writeOpts []litho.WriteOption

	for _, s := range run {
		switch s := s.(type) {
		case *selectStage:
			pages, err := litho.ParsePageRanges(s.Pages, pageCount)
			if err != nil {
				return document{}, &stageError{op: s.op(), filename: d.filename, err: err}
			}

			pipeline.Split(pages)
		case writeOptioner:
			opts, err := s.writeOptions()
			if err != nil {
				return document{}, &stageError{op: s.op(), filename: d.filename, err: err}
			}

			writeOpts = append(writeOpts, opts...)
		case editor:
			pipeline.Apply(func(ctx context.Context, w *litho.Writer) (err error) {
				ctx, span := gotenberg.Tracer().Start(ctx, "pdf."+s.op())
				defer func() {
					_ = gotenberg.SetSpanStatus(span, err, errorType(err))
					span.End()
				}()

				err = s.edit(ctx, w, env)
				if err != nil {
					return &stageError{op: s.op(), filename: d.filename, err: err}
				}

				return nil
			})
		default:
			return document{}, fmt.Errorf("stage %q has no executor", s.op())
		}
	}

	ops := make([]string, len(run))
	for i, s := range run {
		ops[i] = s.op()
	}

	out, err := p.write(ctx, pipeline, ops, d.filename, writeOpts...)
	if err != nil {
		var se *stageError
		if errors.As(err, &se) {
			return document{}, err
		}

		return document{}, &stageError{op: run[len(run)-1].op(), filename: d.filename, err: err}
	}

	if env.report != nil {
		outcome.add(ctx, d.filename, env.report)
	}

	return out, nil
}

// ResidualHeader is the output header counting the violations remediation
// could not fix. The output still claims the profile: a non-zero count means
// it does not conform.
const ResidualHeader = "Gotenberg-Remediation-Residual"

// RolledBackHeader is the output header counting the fixes remediation
// discarded because they changed the content beyond its tolerance.
const RolledBackHeader = "Gotenberg-Remediation-Rolled-Back"

// remediationOutcome sums the remediation reports of the documents of a run.
type remediationOutcome struct {
	mu         sync.Mutex
	residual   int
	rolledBack int
}

// add sums report, and logs what it says of the document named filename: a
// warning when the output falls short of the profile.
func (o *remediationOutcome) add(ctx *api.Context, filename string, report *litho.RemediationReport) {
	o.mu.Lock()
	o.residual += len(report.Residual)
	o.rolledBack += len(report.RolledBack)
	o.mu.Unlock()

	residual := make([]string, len(report.Residual))
	for i, err := range report.Residual {
		residual[i] = err.Error()
	}

	fonts := make([]string, len(report.FontRefusals))
	for i, refusal := range report.FontRefusals {
		fonts[i] = fmt.Sprintf("%s: %s", refusal.BaseFont, refusal.Err)
	}

	attrs := []any{
		slog.String("filename", filename),
		slog.Int("fixed", len(report.Fixed)),
		slog.Any("residual", residual),
		slog.Any("rolled_back", report.RolledBack),
		slog.Any("font_refusals", fonts),
	}

	if len(residual) == 0 && len(report.RolledBack) == 0 && len(fonts) == 0 {
		ctx.Log().DebugContext(ctx, "remediation complete", attrs...)
		return
	}

	ctx.Log().WarnContext(ctx, "remediation incomplete", attrs...)
}

func documentPaths(docs []document) []string {
	paths := make([]string, len(docs))
	for i, d := range docs {
		paths[i] = d.path
	}

	return paths
}

// runEnv is what an edit needs besides the writer: the page count page ranges
// count against, the uploaded files, and the readers to close once the
// pipeline has written.
type runEnv struct {
	processor *Processor
	pageCount int
	files     map[string]string
	readers   []*litho.Reader
	// report is what the remediation of the document said, if any.
	report *litho.RemediationReport
}

// pages resolves an optional page range specification. Empty means every page,
// which litho takes as an empty list.
func (env *runEnv) pages(spec string) ([]int, error) {
	if strings.TrimSpace(spec) == "" {
		return nil, nil
	}

	return litho.ParsePageRanges(spec, env.pageCount)
}

// read returns the content of an uploaded file.
func (env *runEnv) read(name string) ([]byte, error) {
	path, ok := env.files[name]
	if !ok {
		return nil, fmt.Errorf("file %q was not uploaded", name)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", name, err)
	}

	return data, nil
}

// openFile opens an uploaded PDF, kept open until the pipeline has written.
func (env *runEnv) openFile(ctx context.Context, name string) (*litho.Reader, error) {
	path, ok := env.files[name]
	if !ok {
		return nil, fmt.Errorf("file %q was not uploaded", name)
	}

	r, err := env.processor.open(ctx, path, "")
	if err != nil {
		return nil, fmt.Errorf("file %q: %w", name, err)
	}

	env.readers = append(env.readers, r)

	return r, nil
}

func (env *runEnv) close() {
	for _, r := range env.readers {
		_ = r.Close()
	}
}

// editor is a stage that queues edits on a litho writer.
type editor interface {
	stage
	edit(ctx context.Context, w *litho.Writer, env *runEnv) error
}

// writeOptioner is a stage that sets options of the final write.
type writeOptioner interface {
	stage
	writeOptions() ([]litho.WriteOption, error)
}

// mimeTypeOf returns the MIME type of a file from its extension, or the
// generic binary type.
func mimeTypeOf(name string) string {
	t := mime.TypeByExtension(strings.ToLower(filepath.Ext(name)))
	if t == "" {
		return "application/octet-stream"
	}

	mediaType, _, err := mime.ParseMediaType(t)
	if err != nil {
		return t
	}

	return mediaType
}
