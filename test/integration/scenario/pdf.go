package scenario

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gotenberg/litho/pkg/models"
	litho "github.com/gotenberg/litho/pkg/pdf"
)

// The PDF assertions read the produced documents with litho, in process.
// litho's validators are scored against veraPDF in litho's own test suite.

// pdfPath resolves a PDF of the response: a filename in the request's
// directory, or a "*_" pattern matched against the test store.
func (s *scenario) pdfPath(name string) (string, error) {
	if !strings.HasPrefix(name, "*_") {
		path := filepath.Join(s.workdir, s.resp.Header().Get("Gotenberg-Trace"), name)

		_, err := os.Stat(path)
		if err != nil {
			return "", fmt.Errorf("PDF %q does not exist: %w", path, err)
		}

		return path, nil
	}

	substr := strings.TrimPrefix(name, "*_")
	var found string
	err := filepath.Walk(s.teststoreDir, func(path string, info os.FileInfo, pathErr error) error {
		if pathErr != nil {
			return pathErr
		}
		if found == "" && strings.Contains(info.Name(), substr) {
			found = path
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walk %q: %w", s.teststoreDir, err)
	}

	if found == "" {
		return "", fmt.Errorf("no PDF matches %q", name)
	}

	return found, nil
}

// responsePdfPaths returns every PDF of the response or webhook request.
func (s *scenario) responsePdfPaths() ([]string, error) {
	_, err := os.Stat(s.teststoreDir)
	if err != nil {
		return nil, fmt.Errorf("directory %q does not exist: %w", s.teststoreDir, err)
	}

	var paths []string
	err = filepath.Walk(s.teststoreDir, func(path string, info os.FileInfo, pathErr error) error {
		if pathErr != nil {
			return pathErr
		}
		if strings.EqualFold(filepath.Ext(info.Name()), ".pdf") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %q: %w", s.teststoreDir, err)
	}

	return paths, nil
}

// withPdf opens the PDF at path with litho and runs fn on it.
func withPdf(ctx context.Context, path string, fn func(r *litho.Reader) error) error {
	r, err := litho.OpenFile(ctx, path)
	if err != nil {
		return fmt.Errorf("open PDF %q: %w", path, err)
	}
	defer func() { _ = r.Close() }()

	return fn(r)
}

// withResponsePdfs runs fn on every PDF of the response or webhook request.
func (s *scenario) withResponsePdfs(ctx context.Context, fn func(path string, r *litho.Reader) error) error {
	paths, err := s.responsePdfPaths()
	if err != nil {
		return err
	}

	for _, path := range paths {
		err = withPdf(ctx, path, func(r *litho.Reader) error { return fn(path, r) })
		if err != nil {
			return err
		}
	}

	return nil
}

// withNamedPdf runs fn on the PDF name resolves to.
func (s *scenario) withNamedPdf(ctx context.Context, name string, fn func(r *litho.Reader) error) error {
	path, err := s.pdfPath(name)
	if err != nil {
		return err
	}

	return withPdf(ctx, path, fn)
}

func (s *scenario) thePdfsShouldBeValidWithAToleranceOf(ctx context.Context, _, profile string, tolerance int) error {
	return s.withResponsePdfs(ctx, func(path string, r *litho.Reader) error {
		var (
			violations []litho.Violation
			err        error
		)

		switch profile {
		case "PDF/A-1b":
			violations, err = r.ValidatePDFAReport(ctx, models.PDFA1b)
		case "PDF/A-2b":
			violations, err = r.ValidatePDFAReport(ctx, models.PDFA2b)
		case "PDF/A-3b":
			violations, err = r.ValidatePDFAReport(ctx, models.PDFA3b)
		case "PDF/A-4":
			violations, err = r.ValidatePDFAReport(ctx, models.PDFA4)
		case "PDF/UA-1":
			violations, err = r.ValidatePDFUAReport(ctx, models.PDFUA1)
		case "PDF/UA-2":
			violations, err = r.ValidatePDFUAReport(ctx, models.PDFUA2)
		default:
			return fmt.Errorf("unknown profile %q", profile)
		}

		if err != nil {
			return fmt.Errorf("validate %q against %s: %w", path, profile, err)
		}

		// Count failed rules, as veraPDF does, not occurrences.
		failed := make(map[string]struct{})
		for _, v := range violations {
			failed[fmt.Sprintf("%s-%d", v.Clause, v.TestNumber)] = struct{}{}
		}

		if len(failed) > tolerance {
			return fmt.Errorf("expected at most %d failed rule(s) against %s, but %q fails %d: %v", tolerance, profile, path, len(failed), violations)
		}

		return nil
	})
}

func (s *scenario) thePdfShouldHavePages(ctx context.Context, name string, pages int) error {
	return s.withNamedPdf(ctx, name, func(r *litho.Reader) error {
		actual, err := r.PageCount(ctx)
		if err != nil {
			return fmt.Errorf("count pages: %w", err)
		}

		if actual != pages {
			return fmt.Errorf("expected %d pages, but actual is %d", pages, actual)
		}

		return nil
	})
}

// pdfImages returns the images of every page of the named PDF.
func (s *scenario) pdfImages(ctx context.Context, name string) ([][]models.ImageMeta, error) {
	var perPage [][]models.ImageMeta

	err := s.withNamedPdf(ctx, name, func(r *litho.Reader) error {
		pages, err := r.PageCount(ctx)
		if err != nil {
			return fmt.Errorf("count pages: %w", err)
		}

		for page := 1; page <= pages; page++ {
			images, err := r.ExtractImages(ctx, page)
			if err != nil {
				return fmt.Errorf("extract the images of page %d: %w", page, err)
			}

			perPage = append(perPage, images)
		}

		return nil
	})

	return perPage, err
}

func (s *scenario) thePdfShouldHaveImages(ctx context.Context, name string, images int) error {
	perPage, err := s.pdfImages(ctx, name)
	if err != nil {
		return err
	}

	actual := 0
	for _, pageImages := range perPage {
		actual += len(pageImages)
	}

	if actual != images {
		return fmt.Errorf("expected %d image(s), but actual is %d", images, actual)
	}

	return nil
}

func (s *scenario) thePdfShouldReuseImageXObjects(ctx context.Context, name string, images, pages int) error {
	perPage, err := s.pdfImages(ctx, name)
	if err != nil {
		return err
	}

	return checkImageXObjectReuse(perPage, images, pages)
}

// checkImageXObjectReuse checks every page shows images placements, all of the
// same images XObjects.
func checkImageXObjectReuse(perPage [][]models.ImageMeta, images, pages int) error {
	if len(perPage) != pages {
		return fmt.Errorf("expected %d page(s), but actual is %d", pages, len(perPage))
	}

	objects := make(map[int]struct{})
	for i, pageImages := range perPage {
		if len(pageImages) != images {
			return fmt.Errorf("expected %d image appearance(s) on page %d, but actual is %d", images, i+1, len(pageImages))
		}

		for _, img := range pageImages {
			if img.ObjectNum == 0 {
				return fmt.Errorf("an image of page %d is a direct object, which cannot be shared", i+1)
			}
			objects[img.ObjectNum] = struct{}{}
		}
	}

	if len(objects) != images {
		return fmt.Errorf("expected %d unique image XObject(s) shared across %d page(s), but actual is %d", images, pages, len(objects))
	}

	return nil
}

func (s *scenario) thePdfShouldBeSetToLandscapeOrientation(ctx context.Context, name, kind string) error {
	return s.withNamedPdf(ctx, name, func(r *litho.Reader) error {
		page, err := r.Page(ctx, 1)
		if err != nil {
			return fmt.Errorf("read the first page: %w", err)
		}

		if kind == "should NOT" && page.Height < page.Width {
			return fmt.Errorf("expected height %g to be greater than width %g", page.Height, page.Width)
		}

		if kind == "should" && page.Width < page.Height {
			return fmt.Errorf("expected width %g to be greater than height %g", page.Width, page.Height)
		}

		return nil
	})
}

func (s *scenario) thePdfShouldBeTagged(ctx context.Context, name, kind string) error {
	return s.withNamedPdf(ctx, name, func(r *litho.Reader) error {
		tagged, err := r.IsTagged(ctx)
		if err != nil {
			return fmt.Errorf("read tagging: %w", err)
		}

		if want := kind != "should NOT"; tagged != want {
			return fmt.Errorf("expected tagged PDF to be %t, but actual is %t", want, tagged)
		}

		return nil
	})
}

func (s *scenario) thePdfShouldHaveADocumentOutline(ctx context.Context, name, kind string) error {
	return s.withNamedPdf(ctx, name, func(r *litho.Reader) error {
		bookmarks, err := r.Bookmarks(ctx)
		if err != nil {
			return fmt.Errorf("read bookmarks: %w", err)
		}

		hasOutline := len(bookmarks) > 0
		if want := kind != "should NOT"; hasOutline != want {
			return fmt.Errorf("expected a document outline to be %t, but actual is %t", want, hasOutline)
		}

		return nil
	})
}

// pdfPageText extracts the text of a single page of the named PDF.
func (s *scenario) pdfPageText(ctx context.Context, name string, page int) (string, error) {
	var text string

	err := s.withNamedPdf(ctx, name, func(r *litho.Reader) error {
		var err error
		text, err = r.ExtractText(ctx, page)
		if err != nil {
			return fmt.Errorf("extract the text of page %d: %w", page, err)
		}

		return nil
	})

	return text, err
}

func (s *scenario) thePdfsShouldBeFlatten(ctx context.Context, _, should string) error {
	return s.withResponsePdfs(ctx, func(path string, r *litho.Reader) error {
		hasFields, err := r.HasFormFields(ctx)
		if err != nil {
			return fmt.Errorf("read the form fields of %q: %w", path, err)
		}

		if should == "should" && hasFields {
			return fmt.Errorf("PDF %q is not flatten", path)
		}

		if should == "should NOT" && !hasFields {
			return fmt.Errorf("PDF %q is flatten", path)
		}

		return nil
	})
}

func (s *scenario) thePdfsShouldAllowAction(ctx context.Context, _, should, action string) error {
	return s.withResponsePdfs(ctx, func(path string, r *litho.Reader) error {
		permissions := r.Permissions()

		var allowed bool
		switch action {
		case "printing":
			allowed = permissions.Print
		case "copying":
			allowed = permissions.ExtractText
		case "modifying":
			allowed = permissions.ModifyContent
		case "annotating":
			allowed = permissions.ModifyAnnotations
		default:
			return fmt.Errorf("unsupported permission action %q", action)
		}

		if want := should == "should"; allowed != want {
			return fmt.Errorf("expected %q to allow %s to be %t, but actual is %t", path, action, want, allowed)
		}

		return nil
	})
}

func (s *scenario) thePdfsShouldBeEncrypted(ctx context.Context, _, should string) error {
	paths, err := s.responsePdfPaths()
	if err != nil {
		return err
	}

	for _, path := range paths {
		var encrypted bool

		r, err := litho.OpenFile(ctx, path)
		switch {
		case errors.Is(err, litho.ErrInvalidPassword):
			encrypted = true
		case err != nil:
			return fmt.Errorf("open PDF %q: %w", path, err)
		default:
			encrypted = r.IsEncrypted()
			_ = r.Close()
		}

		if want := should == "should"; encrypted != want {
			return fmt.Errorf("expected %q to be encrypted to be %t, but actual is %t", path, want, encrypted)
		}
	}

	return nil
}

// embeddedFiles returns the files embedded in the PDF r reads.
func embeddedFiles(ctx context.Context, path string, r *litho.Reader) ([]models.EmbeddedFile, error) {
	files, err := r.EmbeddedFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("read the embedded files of %q: %w", path, err)
	}

	return files, nil
}

func (s *scenario) thePdfsShouldHaveEmbeddedFile(ctx context.Context, _, should, embed string) error {
	return s.withResponsePdfs(ctx, func(path string, r *litho.Reader) error {
		files, err := embeddedFiles(ctx, path, r)
		if err != nil {
			return err
		}

		found := false
		for _, f := range files {
			found = found || f.Name == embed
		}

		if want := should == "should"; found != want {
			return fmt.Errorf("expected %q to embed %q to be %t, but actual is %t", path, embed, want, found)
		}

		return nil
	})
}

func (s *scenario) thePdfsShouldHaveEmbeddedFileWithRelationship(ctx context.Context, _, embed, relationship string) error {
	return s.withResponsePdfs(ctx, func(path string, r *litho.Reader) error {
		files, err := embeddedFiles(ctx, path, r)
		if err != nil {
			return err
		}

		for _, f := range files {
			if f.Name != embed {
				continue
			}

			if f.Relationship != relationship {
				return fmt.Errorf("expected %q in %q to have relationship %q, but actual is %q", embed, path, relationship, f.Relationship)
			}

			return nil
		}

		return fmt.Errorf("%q does not embed %q", path, embed)
	})
}

func (s *scenario) thePdfsShouldDeclareFacturXConformanceLevel(ctx context.Context, _, conformanceLevel string) error {
	level := regexp.MustCompile(`fx:ConformanceLevel(?:>|=")\s*` + regexp.QuoteMeta(conformanceLevel) + `\s*(?:<|")`)

	return s.withResponsePdfs(ctx, func(path string, r *litho.Reader) error {
		openXMP, err := r.XMPMetadata(ctx)
		if err != nil || openXMP == nil {
			return fmt.Errorf("read the XMP of %q: %w", path, err)
		}

		rc, err := openXMP(ctx)
		if err != nil {
			return fmt.Errorf("open the XMP of %q: %w", path, err)
		}

		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return fmt.Errorf("read the XMP of %q: %w", path, err)
		}

		xmp := string(data)

		if !strings.Contains(xmp, "urn:factur-x:pdfa:CrossIndustryDocument:invoice:1p0#") {
			return fmt.Errorf("missing Factur-X namespace in the XMP of %q", path)
		}

		if !strings.Contains(xmp, "pdfaExtension:schemas") {
			return fmt.Errorf("missing PDF/A extension schema in the XMP of %q", path)
		}

		if !level.MatchString(xmp) {
			return fmt.Errorf("missing fx:ConformanceLevel %q in the XMP of %q", conformanceLevel, path)
		}

		return nil
	})
}

func (s *scenario) thePdfShouldHaveMetadata(ctx context.Context, name, key, expected string) error {
	return s.withNamedPdf(ctx, name, func(r *litho.Reader) error {
		metadata, err := r.Metadata(ctx)
		if err != nil {
			return fmt.Errorf("read metadata: %w", err)
		}

		if metadata[key] != expected {
			return fmt.Errorf("expected metadata %q to be %q, but actual is %q (all: %v)", key, expected, metadata[key], metadata)
		}

		return nil
	})
}

func (s *scenario) thePdfPageShouldBeRotatedBy(ctx context.Context, name string, page, angle int) error {
	return s.withNamedPdf(ctx, name, func(r *litho.Reader) error {
		p, err := r.Page(ctx, page)
		if err != nil {
			return fmt.Errorf("read page %d: %w", page, err)
		}

		if p.Rotate != angle {
			return fmt.Errorf("expected page %d to be rotated by %d, but actual is %d", page, angle, p.Rotate)
		}

		return nil
	})
}

func (s *scenario) thePdfsShouldBeSigned(ctx context.Context, _, should string) error {
	return s.withResponsePdfs(ctx, func(path string, r *litho.Reader) error {
		signatures, err := r.ExtractSignatures(ctx)
		if err != nil {
			return fmt.Errorf("read the signatures of %q: %w", path, err)
		}

		if want := should == "should"; (len(signatures) > 0) != want {
			return fmt.Errorf("expected %q to be signed to be %t, but it has %d signature(s)", path, want, len(signatures))
		}

		return nil
	})
}

func (s *scenario) thePdfShouldHaveSize(ctx context.Context, name, width, height string) error {
	w, err := strconv.ParseFloat(width, 64)
	if err != nil {
		return fmt.Errorf("parse width %q: %w", width, err)
	}

	h, err := strconv.ParseFloat(height, 64)
	if err != nil {
		return fmt.Errorf("parse height %q: %w", height, err)
	}

	return s.withNamedPdf(ctx, name, func(r *litho.Reader) error {
		page, err := r.Page(ctx, 1)
		if err != nil {
			return fmt.Errorf("read the first page: %w", err)
		}

		// One point of tolerance absorbs the rounding of millimeter sizes.
		if abs(page.Width-w) > 1 || abs(page.Height-h) > 1 {
			return fmt.Errorf("expected the first page to be %gx%g points, but actual is %gx%g", w, h, page.Width, page.Height)
		}

		return nil
	})
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}

	return f
}
