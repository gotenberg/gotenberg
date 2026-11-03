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
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/gotenberg/litho/pkg/models"
	litho "github.com/gotenberg/litho/pkg/pdf"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

// Processor runs [Stages] on PDF documents with litho. It is safe for
// concurrent use. The zero value uses litho's default limits and HTTP client,
// and remediates, renders and redacts without the system's fonts.
type Processor struct {
	limits litho.Limits
	fonts  litho.RemediateOption
	// renderFonts draws the fonts a document does not embed, and bounds their
	// glyphs for a redaction. litho bundles no font program: without it, such
	// a font renders as boxes.
	renderFonts litho.RenderOption
	httpClient  func(ctx context.Context) *http.Client
	files       fileConcurrency
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

	// Render comes last, and writes no document: it draws the final ones.
	var render *renderStage
	if len(stages) > 0 {
		if r, ok := stages[len(stages)-1].(*renderStage); ok {
			render = r
			stages = stages[:len(stages)-1]
		}
	}

	// optimize rewrites a document decrypted: only decrypt, which removes the
	// encryption on purpose, or encrypt, which applies one, make that safe. A
	// decrypt stage without a password counts: an owner-password-only PDF
	// opens with an empty user password.
	encryptionHandled := slices.ContainsFunc(stages, func(s stage) bool { return s.op() == "encrypt" || s.op() == "decrypt" })

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
			docs, err = p.runEach(ctx, stages[i:end], docs, password, encryptionHandled, files, outcome)
			i = end
		}

		if err != nil {
			return nil, err
		}

		// Only inputs may be encrypted: decrypt is the first stage, and it
		// removes the encryption from everything it writes.
		password = ""
	}

	var images []string
	if render != nil {
		images, err = p.render(ctx, render, docs, renderPassword(stages, password))
		if err != nil {
			return nil, err
		}
	}

	paths := make([]string, len(docs), len(docs)+len(images))
	for i, d := range docs {
		paths[i] = d.path
	}

	return append(paths, images...), nil
}

// renderPassword returns the password that opens the final documents: the
// one of the encryption the stages apply, or, without stages, the one of the
// inputs.
func renderPassword(stages Stages, inputPassword string) string {
	if len(stages) == 0 {
		return inputPassword
	}

	for _, s := range stages {
		if e, ok := s.(*encryptStage); ok {
			if e.OwnerPassword != "" {
				return e.OwnerPassword
			}

			return e.UserPassword
		}
	}

	return ""
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

// open opens the PDF at path with the processor's limits, and reads its page
// tree: litho opens a truncated or corrupt file lazily, and a route that never
// reads the pages, such as metadata, or a merge of one file, would answer a
// document it cannot read as if it were fine.
func (p *Processor) open(ctx context.Context, path, password string) (*litho.Reader, error) {
	opts := []litho.OpenOption{litho.WithLimits(p.limits)}
	if password != "" {
		opts = append(opts, litho.WithPassword(password))
	}

	r, err := litho.OpenFile(ctx, path, opts...)
	if err != nil {
		return nil, fmt.Errorf("open PDF: %w", err)
	}

	_, err = r.PageCount(ctx)
	if err != nil {
		_ = r.Close()

		return nil, fmt.Errorf("open PDF: count pages: %w", err)
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
	// A single document is not rewritten, but it is read all the same: merge
	// must not hand back a file the other routes refuse.
	if len(docs) == 1 {
		r, err := p.open(ctx, docs[0].path, password)
		if err != nil {
			return nil, &stageError{op: "merge", filename: docs[0].filename, err: err}
		}

		_ = r.Close()
	}

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

		err = refuseSigned(ctx, r)
		if err != nil {
			return nil, &stageError{op: "merge", filename: d.filename, err: err}
		}
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

	err = refuseSigned(ctx, r)
	if err != nil {
		return nil, err
	}

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

	if s.Every != nil {
		every := *s.Every

		for start := 1; start <= pageCount; start += every {
			var pages []int
			for page := start; page < start+every && page <= pageCount; page++ {
				pages = append(pages, page)
			}

			groups = append(groups, pages)
		}

		return groups, nil
	}

	for _, part := range s.Ranges {
		pages, err := parsePages(part, pageCount)
		if err != nil {
			return nil, err
		}

		groups = append(groups, pages)
	}

	return groups, nil
}

// runEach runs a run of stages on every document.
func (p *Processor) runEach(ctx *api.Context, run Stages, docs []document, password string, encryptionHandled bool, files map[string]string, outcome *remediationOutcome) ([]document, error) {
	out := make([]document, len(docs))

	err := p.forEachInputPathIndexed(ctx, documentPaths(docs), func(i int, _ string) error {
		var err error
		out[i], err = p.runOne(ctx, run, docs[i], password, encryptionHandled, files, outcome)

		return err
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

// runOne runs a run of stages on one document through a single litho
// pipeline. encryptionHandled tells that the stages decrypt or encrypt.
func (p *Processor) runOne(ctx *api.Context, run Stages, d document, password string, encryptionHandled bool, files map[string]string, outcome *remediationOutcome) (document, error) {
	r, err := p.open(ctx, d.path, password)
	if err != nil {
		return document{}, &stageError{op: run[0].op(), filename: d.filename, err: err}
	}
	defer func() { _ = r.Close() }()

	pageCount, err := r.PageCount(ctx)
	if err != nil {
		return document{}, &stageError{op: run[0].op(), filename: d.filename, err: fmt.Errorf("count pages: %w", err)}
	}

	// A rewrite drops the signatures of the PDF: refuse it before running.
	// optimize has a guard of its own.
	for _, s := range run {
		if !rewrites(s) {
			continue
		}

		err = refuseSigned(ctx, r)
		if err != nil {
			return document{}, &stageError{op: s.op(), filename: d.filename, err: err}
		}

		break
	}

	env := &runEnv{processor: p, pageCount: pageCount, files: files, source: r}
	defer env.close()

	pipeline := litho.NewPipeline().Open(r)
	var (
		dropped           []litho.DroppedSignature
		writeOpts         = []litho.WriteOption{litho.WithDroppedSignatures(&dropped)}
		optimizing        bool
		compactionSkipped bool
		// litho applies most edits when a later stage or the write runs: a
		// label per editor names the stage that queued a failed edit.
		labels = make(map[string]stage, len(run))
	)

	for i, s := range run {
		switch s := s.(type) {
		case *selectStage:
			pages, err := parsePages(s.Pages, pageCount)
			if err != nil {
				return document{}, &stageError{op: s.op(), filename: d.filename, err: err}
			}

			pipeline.Split(pages)
		case writeOptioner:
			var (
				opts []litho.WriteOption
				err  error
			)

			// optimize rewrites the document, which only the reader tells
			// is safe.
			if o, ok := s.(*optimizeStage); ok {
				opts, err = o.guardedWriteOptions(ctx, r, encryptionHandled)
				opts = append(opts, litho.WithCompactionSkipped(&compactionSkipped))
				optimizing = true
			} else {
				opts, err = s.writeOptions()
			}

			if err != nil {
				return document{}, &stageError{op: s.op(), filename: d.filename, err: err}
			}

			writeOpts = append(writeOpts, opts...)
		case editor:
			label := strconv.Itoa(i)
			labels[label] = s

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
			}, litho.WithEditLabel(label))
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
		// The stage that applied a failed edit, whose error wraps it, is not
		// the one that queued it.
		if e := editError(err); e != nil {
			if s, ok := labels[e.Label]; ok {
				if errors.Is(err, litho.ErrFormFieldNotFound) {
					err = formFieldError(ctx, r, Stages{s}, err)
				}

				return document{}, &stageError{op: s.op(), filename: d.filename, err: err}
			}
		}

		var se *stageError
		if errors.As(err, &se) {
			return document{}, err
		}

		// litho fills the form at write, whatever stage comes last: name the
		// one that asked, and the fields the PDF has instead.
		if errors.Is(err, litho.ErrFormFieldNotFound) {
			return document{}, &stageError{op: "set-form", filename: d.filename, err: formFieldError(ctx, r, run, err)}
		}

		return document{}, &stageError{op: run[len(run)-1].op(), filename: d.filename, err: err}
	}

	// Any edit rewrites a PDF litho repaired on reading: the stages above
	// cannot tell it before running.
	if len(dropped) > 0 {
		return document{}, &stageError{op: run[len(run)-1].op(), filename: d.filename, err: errSignedRewrite()}
	}

	if optimizing && compactionSkipped {
		ctx.Log().WarnContext(ctx, fmt.Sprintf("optimize wrote '%s' without compacting it: litho appended an update to the document rather than rewriting it", d.filename))
	}

	if env.report != nil {
		outcome.add(ctx, d.filename, env.report, env.reportProfiles)
	}

	return out, nil
}

// errSignedRewrite is the refusal of a stage that would rewrite a signed PDF.
func errSignedRewrite() error {
	return &stageInputError{message: "the PDF is signed, and this stage rewrites it, which would remove its signatures"}
}

// refuseSigned returns [errSignedRewrite] when the PDF r reads is signed: a
// signature's digest covers the bytes of the file it was made in (ISO
// 32000-1, 12.8.1), so no rewrite keeps it.
func refuseSigned(ctx context.Context, r *litho.Reader) error {
	signatures, err := r.ExtractSignatures(ctx)
	if err != nil {
		return fmt.Errorf("read the signatures: %w", err)
	}

	if len(signatures) > 0 {
		return errSignedRewrite()
	}

	return nil
}

// rewrites reports whether stage s rewrites the document whole, which drops
// its signatures (litho's Writer.DroppedSignatures lists those edits): select
// and delete build a new file, as merge and split do, and insert, encryption,
// decryption, form flattening, redaction, remediation, color conversion and
// transparency flattening rewrite it. The other edits append an incremental
// update, which keeps them. optimize, which forces a rewrite, has a guard of
// its own.
func rewrites(s stage) bool {
	switch s.(type) {
	case *selectStage, *deleteStage, *insertStage, *encryptStage, *decryptStage, *flattenFormStage,
		*redactStage, *remediateStage, *convertColorsStage, *flattenTransparencyStage:
		return true
	default:
		return false
	}
}

// maxFormFieldsInMessage bounds the field names an unknown field's message
// lists.
const maxFormFieldsInMessage = 20

// formFieldError words err, a [litho.ErrFormFieldNotFound] of a set-form
// stage of run on the document r reads, for the client: the fields the stages
// name that the PDF lacks, and the fields it has.
func formFieldError(ctx context.Context, r *litho.Reader, run Stages, err error) error {
	fields, readErr := r.FormFields(ctx)
	if readErr != nil {
		return fmt.Errorf("%w; read the form fields: %w", err, readErr)
	}

	var names []string
	for _, f := range fields {
		// An XFA form names its fields in its own template, which set-form
		// does not fill.
		if f.Type == models.FormFieldXFA {
			continue
		}

		if !slices.Contains(names, f.Name) {
			names = append(names, f.Name)
		}
	}

	// litho names the first field it misses, in the stages' order: every
	// one the PDF lacks helps more.
	var missing []string
	for _, s := range run {
		set, ok := s.(*setFormStage)
		if !ok {
			continue
		}

		for _, name := range sortedKeys(set.Fields) {
			if !slices.Contains(names, name) && !slices.Contains(missing, name) {
				missing = append(missing, name)
			}
		}
	}

	quoted := make([]string, len(missing))
	for i, name := range missing {
		quoted[i] = strconv.Quote(name)
	}

	var message string
	switch len(quoted) {
	case 0:
		message = "a form field the stage names was not found"
	case 1:
		message = fmt.Sprintf("form field %s not found", quoted[0])
	default:
		message = fmt.Sprintf("form fields %s not found", strings.Join(quoted, ", "))
	}

	switch {
	case len(names) == 0:
		message += "; the PDF has no form fields"
	case len(names) > maxFormFieldsInMessage:
		message += fmt.Sprintf("; the PDF's fields are: %s, and %d more", strings.Join(names[:maxFormFieldsInMessage], ", "), len(names)-maxFormFieldsInMessage)
	default:
		message += "; the PDF's fields are: " + strings.Join(names, ", ")
	}

	return &stageInputError{message: message, err: err}
}

// ResidualHeader is the output header counting the violations remediation
// could not fix, over every profile it remediated to. The output still claims
// each profile: a non-zero count means it does not conform to one of them at
// least.
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

// remediatedProfiles names the profiles a remediation targets: target, the
// PDF/A one or Factur-X, and pdfua, the PDF/UA one, each "" for none.
type remediatedProfiles struct {
	target string
	pdfua  string
}

// of returns the profile a residual violation of the standard spec, a
// [litho.Violation] Specification, belongs to. ISO 14289 states the PDF/UA
// requirements, and ISO 32005 the structure rules PDF/UA-2 applies; every
// other standard belongs to the PDF/A side, the Factur-X rules included.
func (p remediatedProfiles) of(spec string) string {
	if p.target == "" || p.pdfua != "" && (strings.HasPrefix(spec, "ISO_14289") || spec == "ISO_32005") {
		return p.pdfua
	}

	return p.target
}

// add sums report, and logs what it says of the document named filename,
// remediated to profiles: a warning when the output falls short of a profile,
// its residual grouped by profile.
func (o *remediationOutcome) add(ctx *api.Context, filename string, report *litho.RemediationReport, profiles remediatedProfiles) {
	o.mu.Lock()
	o.residual += len(report.Residual)
	o.rolledBack += len(report.RolledBack)
	o.mu.Unlock()

	residual := make(map[string][]string)
	for _, v := range report.ResidualViolations() {
		profile := profiles.of(v.Specification)
		residual[profile] = append(residual[profile], v.Error())
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
	// report is what the remediation of the document said, if any, and
	// reportProfiles the profiles it remediated to.
	report         *litho.RemediationReport
	reportProfiles remediatedProfiles
	// source is the document the run edits, and attachments the attach and
	// detach stages the run applied to it so far, in order.
	source      *litho.Reader
	attachments []attachmentChange
}

// attachmentChange is a file an attach stage embeds, or a detach stage
// removes.
type attachmentChange struct {
	name  string
	added bool
}

// attachmentNames returns the names of the files the document embeds as the
// run has staged it: the source's, with the run's attach and detach stages so
// far applied. The writer stages them, so the source alone is out of date.
func (env *runEnv) attachmentNames(ctx context.Context) ([]string, error) {
	files, err := env.source.EmbeddedFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("read the embedded files: %w", err)
	}

	var names []string
	for _, f := range files {
		if !slices.Contains(names, f.Name) {
			names = append(names, f.Name)
		}
	}

	for _, change := range env.attachments {
		names = slices.DeleteFunc(names, func(name string) bool { return name == change.name })
		if change.added {
			names = append(names, change.name)
		}
	}

	return names, nil
}

// pages resolves an optional page range specification. Empty means every page,
// which litho takes as an empty list.
func (env *runEnv) pages(spec string) ([]int, error) {
	if strings.TrimSpace(spec) == "" {
		return nil, nil
	}

	return parsePages(spec, env.pageCount)
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
