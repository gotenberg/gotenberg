package pdf

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gotenberg/litho/pkg/models"
)

// mergeStage appends every current document, in order, into one.
type mergeStage struct{}

func (*mergeStage) op() string      { return "merge" }
func (*mergeStage) validate() error { return nil }
func (*mergeStage) files() []string { return nil }

// splitStage turns each current document into several. Every splits it into
// chunks of that many pages; Ranges into one document per page selection, such
// as ["1-3","4","5,7"].
type splitStage struct {
	Every  int      `json:"every"`
	Ranges []string `json:"ranges"`
}

func (*splitStage) op() string { return "split" }

func (s *splitStage) validate() error {
	switch {
	case s.Every == 0 && len(s.Ranges) == 0:
		return errors.New(`set "every" or "ranges"`)
	case s.Every != 0 && len(s.Ranges) != 0:
		return errors.New(`set "every" or "ranges", not both`)
	case s.Every < 0:
		return fmt.Errorf(`"every" must be a positive number of pages, got %d`, s.Every)
	}

	for i, r := range s.Ranges {
		if strings.TrimSpace(r) == "" {
			return fmt.Errorf(`"ranges" entry %d must select pages, such as "1-3"`, i)
		}
	}

	return nil
}

func (*splitStage) files() []string { return nil }

// selectStage keeps the given pages, in the given order.
type selectStage struct {
	Pages string `json:"pages"`
}

func (*selectStage) op() string        { return "select" }
func (s *selectStage) validate() error { return requirePages(s.Pages) }
func (*selectStage) files() []string   { return nil }

// deleteStage removes the given pages.
type deleteStage struct {
	Pages string `json:"pages"`
}

func (*deleteStage) op() string        { return "delete" }
func (s *deleteStage) validate() error { return requirePages(s.Pages) }
func (*deleteStage) files() []string   { return nil }

// insertStage inserts the pages of an uploaded PDF after page At, or before
// the first page when At is 0. SourcePages selects the inserted pages, all of
// them when empty.
type insertStage struct {
	Source      string `json:"source"`
	At          int    `json:"at"`
	SourcePages string `json:"sourcePages"`
}

func (*insertStage) op() string { return "insert" }

func (s *insertStage) validate() error {
	if s.Source == "" {
		return errors.New(`"source" must name an uploaded PDF`)
	}

	if s.At < 0 {
		return fmt.Errorf(`"at" must be 0 or a page number, got %d`, s.At)
	}

	return nil
}

func (s *insertStage) files() []string { return []string{s.Source} }

// rotateStage sets the rotation of the given pages, all of them when Pages is
// empty.
type rotateStage struct {
	Angle int    `json:"angle"`
	Pages string `json:"pages"`
}

func (*rotateStage) op() string { return "rotate" }

func (s *rotateStage) validate() error {
	if s.Angle%90 != 0 {
		return fmt.Errorf(`"angle" must be a multiple of 90, got %d`, s.Angle)
	}

	return nil
}

func (*rotateStage) files() []string { return nil }

// stampTextStage draws a line of text over the given pages, all of them when
// Pages is empty. Coordinates are in points from the bottom-left corner.
type stampTextStage struct {
	Text     string  `json:"text"`
	Pages    string  `json:"pages"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	FontSize float64 `json:"fontSize"`
	Font     string  `json:"font"`
	Color    string  `json:"color"`
	Opacity  float64 `json:"opacity"`
	Rotation float64 `json:"rotation"`
}

func (*stampTextStage) op() string { return "stamp-text" }

func (s *stampTextStage) validate() error {
	if s.Text == "" {
		return errors.New(`"text" must not be empty`)
	}

	if s.Font != "" {
		_, ok := stampFonts[s.Font]
		if !ok {
			return fmt.Errorf(`"font" must be one of the 14 standard PDF fonts, such as Helvetica or Times-Roman, got %q`, s.Font)
		}
	}

	if s.Color != "" {
		_, err := parseColor(s.Color)
		if err != nil {
			return err
		}
	}

	return validateOpacity(s.Opacity)
}

func (*stampTextStage) files() []string { return nil }

// stampImageStage draws an uploaded PNG or JPEG image over the given pages,
// all of them when Pages is empty.
type stampImageStage struct {
	Image    string  `json:"image"`
	Pages    string  `json:"pages"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Opacity  float64 `json:"opacity"`
	Rotation float64 `json:"rotation"`
}

func (*stampImageStage) op() string { return "stamp-image" }

func (s *stampImageStage) validate() error {
	if s.Image == "" {
		return errors.New(`"image" must name an uploaded PNG or JPEG file`)
	}

	_, err := imageMimeType(s.Image)
	if err != nil {
		return err
	}

	return validateOpacity(s.Opacity)
}

func (s *stampImageStage) files() []string { return []string{s.Image} }

// stampPdfStage draws page SourcePage of the uploaded PDF Source over the
// given pages, all of them when Pages is empty.
type stampPdfStage struct {
	Source     string  `json:"source"`
	SourcePage int     `json:"sourcePage"`
	Pages      string  `json:"pages"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Scale      float64 `json:"scale"`
	Opacity    float64 `json:"opacity"`
	Rotation   float64 `json:"rotation"`
}

func (*stampPdfStage) op() string { return "stamp-pdf" }

func (s *stampPdfStage) validate() error {
	if s.Source == "" {
		return errors.New(`"source" must name an uploaded PDF`)
	}

	if s.SourcePage < 0 {
		return fmt.Errorf(`"sourcePage" must be a page number, got %d`, s.SourcePage)
	}

	return validateOpacity(s.Opacity)
}

func (s *stampPdfStage) files() []string { return []string{s.Source} }

// setMetadataStage writes the document information. Empty fields are left
// out.
type setMetadataStage struct {
	Title        string `json:"title"`
	Author       string `json:"author"`
	Subject      string `json:"subject"`
	Keywords     string `json:"keywords"`
	Creator      string `json:"creator"`
	Producer     string `json:"producer"`
	CreationDate string `json:"creationDate"`
	ModDate      string `json:"modDate"`
}

func (*setMetadataStage) op() string      { return "set-metadata" }
func (*setMetadataStage) validate() error { return nil }
func (*setMetadataStage) files() []string { return nil }

// setBookmarksStage replaces the outline.
type setBookmarksStage struct {
	Bookmarks []models.Bookmark `json:"bookmarks"`
}

func (*setBookmarksStage) op() string      { return "set-bookmarks" }
func (*setBookmarksStage) validate() error { return nil }
func (*setBookmarksStage) files() []string { return nil }

// setFormStage fills form fields, keyed by their fully qualified name.
type setFormStage struct {
	Fields map[string]string `json:"fields"`
}

func (*setFormStage) op() string { return "set-form" }

func (s *setFormStage) validate() error {
	if len(s.Fields) == 0 {
		return errors.New(`"fields" must map at least one field name to a value`)
	}

	return nil
}

func (*setFormStage) files() []string { return nil }

// flattenFormStage burns the form fields into the page content.
type flattenFormStage struct{}

func (*flattenFormStage) op() string      { return "flatten-form" }
func (*flattenFormStage) validate() error { return nil }
func (*flattenFormStage) files() []string { return nil }

// flattenTransparencyStage replaces transparent content with opaque content.
type flattenTransparencyStage struct{}

func (*flattenTransparencyStage) op() string      { return "flatten-transparency" }
func (*flattenTransparencyStage) validate() error { return nil }
func (*flattenTransparencyStage) files() []string { return nil }

// convertColorsStage converts the colors to one color space: "cmyk", "rgb",
// "gray", or "output-intent" for the document's own output intent.
type convertColorsStage struct {
	Target string `json:"target"`
}

func (*convertColorsStage) op() string { return "convert-colors" }

func (s *convertColorsStage) validate() error {
	switch s.Target {
	case "cmyk", "rgb", "gray", "output-intent":
		return nil
	default:
		return fmt.Errorf(`"target" must be cmyk, rgb, gray or output-intent, got %q`, s.Target)
	}
}

func (*convertColorsStage) files() []string { return nil }

// redactRegion is a rectangle to redact, in points from the bottom-left
// corner of a 1-based page.
type redactRegion struct {
	Page   int     `json:"page"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// redactStage removes the content under the given regions.
type redactStage struct {
	Regions []redactRegion `json:"regions"`
}

func (*redactStage) op() string { return "redact" }

func (s *redactStage) validate() error {
	if len(s.Regions) == 0 {
		return errors.New(`"regions" must list at least one region`)
	}

	for i, r := range s.Regions {
		if r.Page < 1 || r.Width <= 0 || r.Height <= 0 {
			return fmt.Errorf(`region %d needs a "page" of 1 or more and a positive "width" and "height"`, i)
		}
	}

	return nil
}

func (*redactStage) files() []string { return nil }

// autotagStage infers a structure tree for an untagged document.
type autotagStage struct{}

func (*autotagStage) op() string      { return "autotag" }
func (*autotagStage) validate() error { return nil }
func (*autotagStage) files() []string { return nil }

// attachStage embeds an uploaded file. A Relationship ("Source", "Data",
// "Alternative", "Supplement" or "Unspecified") makes it a PDF/A-3 associated
// file.
type attachStage struct {
	File         string `json:"file"`
	Name         string `json:"name"`
	MimeType     string `json:"mimeType"`
	Relationship string `json:"relationship"`
}

func (*attachStage) op() string { return "attach" }

func (s *attachStage) validate() error {
	if s.File == "" {
		return errors.New(`"file" must name an uploaded file`)
	}

	switch s.Relationship {
	case "", "Source", "Data", "Alternative", "Supplement", "Unspecified":
		return nil
	default:
		return fmt.Errorf(`"relationship" must be Source, Data, Alternative, Supplement or Unspecified, got %q`, s.Relationship)
	}
}

func (s *attachStage) files() []string { return []string{s.File} }

// detachStage removes an embedded file by name.
type detachStage struct {
	Name string `json:"name"`
}

func (*detachStage) op() string { return "detach" }

func (s *detachStage) validate() error {
	if s.Name == "" {
		return errors.New(`"name" must name an embedded file`)
	}

	return nil
}

func (*detachStage) files() []string { return nil }

// optimizeStage shrinks the output. Every option defaults to true.
type optimizeStage struct {
	ObjectStreams    *bool `json:"objectStreams"`
	Dedup            *bool `json:"dedup"`
	RecompressImages *bool `json:"recompressImages"`
	SubsetFonts      *bool `json:"subsetFonts"`
}

func (*optimizeStage) op() string      { return "optimize" }
func (*optimizeStage) validate() error { return nil }
func (*optimizeStage) files() []string { return nil }

// permissions lists what an encrypted document allows. Every permission
// defaults to true.
type permissions struct {
	Print             *bool `json:"print"`
	PrintHighQuality  *bool `json:"printHighQuality"`
	ModifyContent     *bool `json:"modifyContent"`
	ExtractText       *bool `json:"extractText"`
	ModifyAnnotations *bool `json:"modifyAnnotations"`
	FillForms         *bool `json:"fillForms"`
	Assemble          *bool `json:"assemble"`
}

// encryptStage protects the output with passwords.
type encryptStage struct {
	UserPassword  string       `json:"userPassword"`
	OwnerPassword string       `json:"ownerPassword"`
	Algorithm     string       `json:"algorithm"`
	Permissions   *permissions `json:"permissions"`
}

func (*encryptStage) op() string { return "encrypt" }

func (s *encryptStage) validate() error {
	if s.UserPassword == "" && s.OwnerPassword == "" {
		return errors.New(`set "userPassword", "ownerPassword", or both`)
	}

	_, ok := encryptAlgorithms[s.Algorithm]
	if !ok {
		return fmt.Errorf(`"algorithm" must be aes256, aes128, rc4-128 or rc4-40, got %q`, s.Algorithm)
	}

	return nil
}

func (*encryptStage) files() []string { return nil }

// decryptStage opens encrypted inputs with Password and removes their
// encryption.
type decryptStage struct {
	Password string `json:"password"`
}

func (*decryptStage) op() string      { return "decrypt" }
func (*decryptStage) validate() error { return nil }
func (*decryptStage) files() []string { return nil }

// remediateStage repairs the document to a conformance profile: a PDF/A or
// PDF/UA one, or Factur-X, which embeds the invoice XML into a PDF/A-3
// document. Without the XML, Factur-X repairs the invoice the document
// already carries, a Factur-X or ZUGFeRD one, leaving its XML as is. Aggressive allows the fixes that may change how pages look, and
// FigureAltText fills the alternate text of the figures that lack one.
type remediateStage struct {
	Profile       string `json:"profile"`
	Language      string `json:"language"`
	Aggressive    bool   `json:"aggressive"`
	FigureAltText string `json:"figureAltText"`
	Xml           string `json:"xml"`
	Level         string `json:"level"`
	DocumentType  string `json:"documentType"`
}

func (*remediateStage) op() string { return "remediate" }

func (s *remediateStage) validate() error {
	_, isPdfa := pdfaProfiles[s.Profile]
	_, isPdfua := pdfuaProfiles[s.Profile]

	switch {
	case isPdfa || isPdfua:
		if s.Xml != "" || s.Level != "" || s.DocumentType != "" {
			return errors.New(`"xml", "level" and "documentType" only apply to the Factur-X profile`)
		}
	case s.Profile == facturXProfile:
		if s.Xml == "" {
			if s.Level != "" || s.DocumentType != "" {
				return errors.New(`"level" and "documentType" need "xml", the uploaded Factur-X invoice XML to embed; without "xml", the remediation repairs the invoice the PDF already carries`)
			}

			return nil
		}

		_, ok := facturXLevels[s.Level]
		if !ok {
			return fmt.Errorf(`"level" must be one of %s, got %q`, strings.Join(sortedKeys(facturXLevels), ", "), s.Level)
		}
	default:
		return fmt.Errorf(`"profile" must be one of %s, PDF/UA-1, PDF/UA-2 or %s, got %q`, strings.Join(sortedKeys(pdfaProfiles), ", "), facturXProfile, s.Profile)
	}

	return nil
}

func (s *remediateStage) files() []string {
	if s.Xml == "" {
		return nil
	}

	return []string{s.Xml}
}

// signStage signs the output with an uploaded PKCS#12 file, or a PEM
// certificate and key. CertificatePassword opens the PKCS#12 file.
// TimestampUrl embeds a timestamp from that authority.
type signStage struct {
	Certificate         string `json:"certificate"`
	Key                 string `json:"key"`
	CertificatePassword string `json:"certificatePassword"`
	Reason              string `json:"reason"`
	Location            string `json:"location"`
	ContactInfo         string `json:"contactInfo"`
	Name                string `json:"name"`
	Pades               bool   `json:"pades"`
	TimestampUrl        string `json:"timestampUrl"`
}

func (*signStage) op() string { return "sign" }

func (s *signStage) validate() error {
	if s.Certificate == "" {
		return errors.New(`"certificate" must name an uploaded PKCS#12 file, or a PEM certificate with "key"`)
	}

	if s.TimestampUrl != "" {
		return validateHttpUrl("timestampUrl", s.TimestampUrl)
	}

	return nil
}

func (s *signStage) files() []string {
	if s.Key == "" {
		return []string{s.Certificate}
	}

	return []string{s.Certificate, s.Key}
}

// timestampStage adds a document timestamp from an RFC 3161 authority.
type timestampStage struct {
	TimestampUrl string `json:"timestampUrl"`
}

func (*timestampStage) op() string { return "timestamp" }
func (s *timestampStage) validate() error {
	return validateHttpUrl("timestampUrl", s.TimestampUrl)
}
func (*timestampStage) files() []string { return nil }

// renderStage draws the pages of each final document as images, and adds them
// to the outputs next to the document. Pages lists the pages to draw, every
// page when empty. Dpi defaults to litho's [litho.DefaultRenderDPI], Format to
// png, and Quality, for jpeg only, to 100.
type renderStage struct {
	Pages   string `json:"pages"`
	Dpi     int    `json:"dpi"`
	Format  string `json:"format"`
	Quality int    `json:"quality"`
}

func (*renderStage) op() string { return "render" }

func (s *renderStage) validate() error {
	if s.Dpi < 0 {
		return fmt.Errorf(`"dpi" must be positive, got %d`, s.Dpi)
	}

	switch s.Format {
	case "", "png":
		if s.Quality != 0 {
			return errors.New(`"quality" only applies to the jpeg format`)
		}
	case "jpeg":
		if s.Quality < 0 || s.Quality > 100 {
			return fmt.Errorf(`"quality" must be between 1 and 100, got %d`, s.Quality)
		}
	default:
		return fmt.Errorf(`"format" must be png or jpeg, got %q`, s.Format)
	}

	return nil
}

func (*renderStage) files() []string { return nil }

// Interface guards.
var (
	_ stage = (*mergeStage)(nil)
	_ stage = (*splitStage)(nil)
	_ stage = (*selectStage)(nil)
	_ stage = (*deleteStage)(nil)
	_ stage = (*insertStage)(nil)
	_ stage = (*rotateStage)(nil)
	_ stage = (*stampTextStage)(nil)
	_ stage = (*stampImageStage)(nil)
	_ stage = (*stampPdfStage)(nil)
	_ stage = (*setMetadataStage)(nil)
	_ stage = (*setBookmarksStage)(nil)
	_ stage = (*setFormStage)(nil)
	_ stage = (*flattenFormStage)(nil)
	_ stage = (*flattenTransparencyStage)(nil)
	_ stage = (*convertColorsStage)(nil)
	_ stage = (*redactStage)(nil)
	_ stage = (*autotagStage)(nil)
	_ stage = (*attachStage)(nil)
	_ stage = (*detachStage)(nil)
	_ stage = (*optimizeStage)(nil)
	_ stage = (*encryptStage)(nil)
	_ stage = (*decryptStage)(nil)
	_ stage = (*remediateStage)(nil)
	_ stage = (*signStage)(nil)
	_ stage = (*timestampStage)(nil)
	_ stage = (*renderStage)(nil)
)
