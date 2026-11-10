package pdf

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/gotenberg/litho/pkg/models"

	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
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
	Every  *int     `json:"every"`
	Ranges []string `json:"ranges"`
}

func (*splitStage) op() string { return "split" }

func (s *splitStage) validate() error {
	switch {
	case s.Every == nil && len(s.Ranges) == 0:
		return errors.New(`set "every" or "ranges"`)
	case s.Every != nil && len(s.Ranges) != 0:
		return errors.New(`set "every" or "ranges", not both`)
	case s.Every != nil && *s.Every < 1:
		return fmt.Errorf(`"every" must be at least 1, got %d`, *s.Every)
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
	Angle *int   `json:"angle"`
	Pages string `json:"pages"`
}

func (*rotateStage) op() string { return "rotate" }

func (s *rotateStage) validate() error {
	if s.Angle == nil {
		return errors.New(`"angle" is required, a multiple of 90 such as 90, 180 or 270`)
	}

	if *s.Angle%90 != 0 {
		return fmt.Errorf(`"angle" must be a multiple of 90, got %d`, *s.Angle)
	}

	return nil
}

func (*rotateStage) files() []string { return nil }

// stampTextStage draws a line of text over the given pages, all of them when
// Pages is empty. Coordinates are in points from the bottom-left corner, or
// offsets from Position when it is set.
type stampTextStage struct {
	Text     string   `json:"text"`
	Pages    string   `json:"pages"`
	X        float64  `json:"x"`
	Y        float64  `json:"y"`
	Position string   `json:"position"`
	FontSize *float64 `json:"fontSize"`
	Font     string   `json:"font"`
	Color    string   `json:"color"`
	Opacity  float64  `json:"opacity"`
	Rotation float64  `json:"rotation"`
}

func (*stampTextStage) op() string { return "stamp-text" }

func (s *stampTextStage) validate() error {
	if s.Text == "" {
		return errors.New(`"text" must not be empty`)
	}

	if s.FontSize != nil && *s.FontSize <= 0 {
		return fmt.Errorf(`"fontSize" must be greater than 0, got %g`, *s.FontSize)
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

	_, err := parsePosition(s.Position)
	if err != nil {
		return err
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
	Position string  `json:"position"`
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

	_, err = parsePosition(s.Position)
	if err != nil {
		return err
	}

	return validateOpacity(s.Opacity)
}

func (s *stampImageStage) files() []string { return []string{s.Image} }

// stampPdfStage draws page SourcePage of the uploaded PDF Source over the
// given pages, all of them when Pages is empty.
type stampPdfStage struct {
	Source     string   `json:"source"`
	SourcePage int      `json:"sourcePage"`
	Pages      string   `json:"pages"`
	X          float64  `json:"x"`
	Y          float64  `json:"y"`
	Position   string   `json:"position"`
	Scale      *float64 `json:"scale"`
	Opacity    float64  `json:"opacity"`
	Rotation   float64  `json:"rotation"`
}

func (*stampPdfStage) op() string { return "stamp-pdf" }

func (s *stampPdfStage) validate() error {
	if s.Source == "" {
		return errors.New(`"source" must name an uploaded PDF`)
	}

	if s.SourcePage < 0 {
		return fmt.Errorf(`"sourcePage" must be a page number, got %d`, s.SourcePage)
	}

	if s.Scale != nil && *s.Scale <= 0 {
		return fmt.Errorf(`"scale" must be greater than 0, got %g`, *s.Scale)
	}

	_, err := parsePosition(s.Position)
	if err != nil {
		return err
	}

	return validateOpacity(s.Opacity)
}

func (s *stampPdfStage) files() []string { return []string{s.Source} }

// composeStage draws pages of the current documents on new pages of a chosen
// size, and replaces them with that one document.
type composeStage struct {
	PaperSize   string      `json:"paperSize"`
	PaperWidth  string      `json:"paperWidth"`
	PaperHeight string      `json:"paperHeight"`
	Placements  []placement `json:"placements"`

	// width and height hold the page size, in inches, once validate resolved
	// it.
	width  float64
	height float64
}

// placement draws a page of one of the current documents on a composed page.
type placement struct {
	Source     string   `json:"source"`
	SourcePage int      `json:"sourcePage"`
	Page       int      `json:"page"`
	X          float64  `json:"x"`
	Y          float64  `json:"y"`
	Scale      *float64 `json:"scale"`
	Rotation   float64  `json:"rotation"`
}

func (*composeStage) op() string { return "compose" }

// validate resolves the page size: PaperSize sets both sides, and PaperWidth
// or PaperHeight then override one at a time, as on the conversion routes. It
// leaves the range litho allows, 3 to 14,400 points a side, to litho.
func (s *composeStage) validate() error {
	s.width, s.height = 0, 0

	if s.PaperSize != "" {
		width, height, ok := api.PaperSizeInches(s.PaperSize)
		if !ok {
			return fmt.Errorf(`"paperSize" must be A3, A4, A5, B4, B5, Letter, Legal or Tabloid, got %q`, s.PaperSize)
		}

		s.width, s.height = width, height
	}

	for _, side := range []struct {
		name   string
		value  string
		target *float64
	}{
		{name: "paperWidth", value: s.PaperWidth, target: &s.width},
		{name: "paperHeight", value: s.PaperHeight, target: &s.height},
	} {
		if side.value == "" {
			continue
		}

		inches, err := api.ParseInches(side.value)
		if err != nil {
			return fmt.Errorf(`%q must be a length such as "6in", "15cm" or "432pt", got %q`, side.name, side.value)
		}

		if inches <= 0 {
			return fmt.Errorf(`%q must be greater than 0, got %q`, side.name, side.value)
		}

		*side.target = inches
	}

	if s.width == 0 || s.height == 0 {
		return errors.New(`set "paperSize", or "paperWidth" and "paperHeight"`)
	}

	if len(s.Placements) == 0 {
		return errors.New(`"placements" must list at least one placement`)
	}

	for i, pl := range s.Placements {
		switch {
		case pl.SourcePage < 0:
			return fmt.Errorf(`"placements" entry %d: "sourcePage" must be a page number, got %d`, i, pl.SourcePage)
		case pl.Page < 0:
			return fmt.Errorf(`"placements" entry %d: "page" must be a page number, got %d`, i, pl.Page)
		case pl.Scale != nil && *pl.Scale <= 0:
			return fmt.Errorf(`"placements" entry %d: "scale" must be greater than 0, got %g`, i, *pl.Scale)
		}
	}

	return nil
}

// files returns nil: the placements name the current documents, which stay
// the PDFs to process, where a stamp-pdf or insert source is an upload the
// stage consumes.
func (*composeStage) files() []string { return nil }

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

func (*setMetadataStage) op() string { return "set-metadata" }

func (s *setMetadataStage) validate() error {
	err := validatePDFDate("creationDate", s.CreationDate)
	if err != nil {
		return err
	}

	return validatePDFDate("modDate", s.ModDate)
}

func (*setMetadataStage) files() []string { return nil }

// setBookmarksStage replaces the outline.
type setBookmarksStage struct {
	Bookmarks []models.Bookmark `json:"bookmarks"`
}

func (*setBookmarksStage) op() string { return "set-bookmarks" }

func (s *setBookmarksStage) validate() error {
	if len(s.Bookmarks) == 0 {
		return errors.New(`"bookmarks" must list at least one bookmark, such as [{"title":"Intro","pageNumber":1}]`)
	}

	return validateBookmarkTitles(s.Bookmarks, "bookmarks")
}

func (*setBookmarksStage) files() []string { return nil }

// validateBookmarkTitles checks that every bookmark, children included, has a
// title. path names the list in the message, such as "bookmarks[1].children".
func validateBookmarkTitles(bookmarks []models.Bookmark, path string) error {
	for i, b := range bookmarks {
		at := fmt.Sprintf("%s[%d]", path, i)

		if strings.TrimSpace(b.Title) == "" {
			return fmt.Errorf(`%s.title is required`, at)
		}

		err := validateBookmarkTitles(b.Children, at+".children")
		if err != nil {
			return err
		}
	}

	return nil
}

// validateBookmarkPages checks that every bookmark, children included, leads
// to a page of a document of pageCount pages. path names the list in the
// message, such as "bookmarks[1].children".
func validateBookmarkPages(bookmarks []models.Bookmark, path string, pageCount int) error {
	for i, b := range bookmarks {
		at := fmt.Sprintf("%s[%d]", path, i)

		switch {
		case b.PageNumber > pageCount:
			return &stageInputError{message: fmt.Sprintf("%s.pageNumber %d is past the last page %d", at, b.PageNumber, pageCount)}
		case b.PageNumber < 1:
			return &stageInputError{message: fmt.Sprintf("%s.pageNumber must be a page from 1 to %d, got %d", at, pageCount, b.PageNumber)}
		}

		err := validateBookmarkPages(b.Children, at+".children", pageCount)
		if err != nil {
			return err
		}
	}

	return nil
}

// setFormStage fills form fields, keyed by their fully qualified name. A value
// is a string, or, for a list box that selects several, an array of strings.
type setFormStage struct {
	Fields map[string]json.RawMessage `json:"fields"`

	// texts and selections hold the values of Fields, once validate decoded
	// them: the strings, and the arrays.
	texts      map[string]string
	selections map[string][]string
}

func (*setFormStage) op() string { return "set-form" }

func (s *setFormStage) validate() error {
	if len(s.Fields) == 0 {
		return errors.New(`"fields" must map at least one field name to a value`)
	}

	s.texts = make(map[string]string, len(s.Fields))
	s.selections = make(map[string][]string)

	for name, raw := range s.Fields {
		var text string
		if json.Unmarshal(raw, &text) == nil {
			s.texts[name] = text
			continue
		}

		var selections []string
		if json.Unmarshal(raw, &selections) != nil {
			return errors.New(`"fields" values must be strings, or arrays of strings for a list that selects several; a check box takes its state name, such as "Yes" or "Off"`)
		}

		s.selections[name] = selections
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

// extractAttachmentsStage writes the files each final document embeds, every
// one or the one Name gives, next to the outputs. It comes last, and writes
// no document.
type extractAttachmentsStage struct {
	Name string `json:"name"`
}

func (*extractAttachmentsStage) op() string { return "extract-attachments" }

func (*extractAttachmentsStage) validate() error { return nil }

func (*extractAttachmentsStage) files() []string { return nil }

// extractTextStage writes the text each final document shows as a .txt file
// next to the outputs, its pages joined by a form feed. Pages lists the pages
// to read, every page when empty. Invisible adds the text a searchable scan
// carries under its image, Artifacts the headers, footers and page numbers,
// and Annotations the text annotations show. It comes last, and writes no
// document.
type extractTextStage struct {
	Pages       string `json:"pages"`
	Invisible   bool   `json:"invisible"`
	Artifacts   bool   `json:"artifacts"`
	Annotations bool   `json:"annotations"`
}

func (*extractTextStage) op() string { return "extract-text" }

func (*extractTextStage) validate() error { return nil }

func (*extractTextStage) files() []string { return nil }

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

// permissionNames are the keys of [permissions], in the order it declares them.
var permissionNames = []string{"print", "printHighQuality", "modifyContent", "extractText", "modifyAnnotations", "fillForms", "assemble"}

// UnmarshalJSON decodes the permissions, and names the valid keys when one is
// unknown: encoding/json would quote its own text, which lists none.
func (p *permissions) UnmarshalJSON(data []byte) error {
	var keys map[string]json.RawMessage
	err := json.Unmarshal(data, &keys)
	if err != nil {
		return err
	}

	for _, key := range sortedKeys(keys) {
		if !slices.Contains(permissionNames, key) {
			return fmt.Errorf(`"permissions" has an unknown key %q; the keys are %s`, key, joinAnd(permissionNames))
		}
	}

	type plain permissions

	return json.Unmarshal(data, (*plain)(p))
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
// already carries, a Factur-X or ZUGFeRD one, leaving its XML as is. Profile
// may also pair a PDF/A profile, or Factur-X, with the PDF/UA profile of its
// PDF version, and the document then conforms to both. Aggressive allows the
// fixes that may change how pages look, and FigureAltText fills the alternate
// text of the figures that lack one.
type remediateStage struct {
	Profile       profileSet `json:"profile"`
	Language      string     `json:"language"`
	Aggressive    bool       `json:"aggressive"`
	FigureAltText string     `json:"figureAltText"`
	Xml           string     `json:"xml"`
	Level         string     `json:"level"`
	DocumentType  string     `json:"documentType"`
}

func (*remediateStage) op() string { return "remediate" }

func (s *remediateStage) validate() error {
	target, _, err := s.Profile.split()
	if err != nil {
		return err
	}

	if target != facturXProfile {
		if s.Xml != "" || s.Level != "" || s.DocumentType != "" {
			return errors.New(`"xml", "level" and "documentType" only apply to the Factur-X profile`)
		}

		return nil
	}

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

	return nil
}

// profileSet is the "profile" option of the remediate stage and of the validate
// route: one profile, sent as a JSON string, or a PDF/A profile, or Factur-X,
// paired with the PDF/UA profile of its PDF version, sent as a JSON array of
// the two in any order.
type profileSet []string

// UnmarshalJSON decodes a JSON string as one profile, and a JSON array of
// strings as several.
func (p *profileSet) UnmarshalJSON(data []byte) error {
	var one string
	if json.Unmarshal(data, &one) == nil {
		*p = profileSet{one}

		return nil
	}

	var many []string
	if json.Unmarshal(data, &many) != nil {
		return errors.New(`"profile" must be a string, or a JSON array of two strings that pair a PDF/A profile with a PDF/UA one`)
	}

	*p = many

	return nil
}

// MarshalJSON encodes one profile as a JSON string and a pair as a JSON array,
// the way the client sent them.
func (p profileSet) MarshalJSON() ([]byte, error) {
	if len(p) == 1 {
		return json.Marshal(p[0])
	}

	return json.Marshal([]string(p))
}

// String returns p as JSON: a quoted profile, an array of them, or "" for
// none.
func (p profileSet) String() string {
	if len(p) == 0 {
		return `""`
	}

	b, err := p.MarshalJSON()
	if err != nil {
		return fmt.Sprint([]string(p))
	}

	return string(b)
}

// split returns the PDF/A-based profile p names, a PDF/A one or Factur-X, and
// its PDF/UA one, "" for none. A profile no stage takes, or a pair litho
// cannot remediate or validate together, returns an error that lists the
// profiles and the pairs.
func (p profileSet) split() (target, pdfua string, err error) {
	switch len(p) {
	case 1:
		switch {
		case pdfaProfiles[p[0]] != "", p[0] == facturXProfile:
			return p[0], "", nil
		case pdfuaProfiles[p[0]] != "":
			return "", p[0], nil
		}
	case 2:
		a, ua := p[0], p[1]
		if pdfuaProfiles[a] != "" {
			a, ua = ua, a
		}

		if pdfuaProfiles[ua] != "" && pairedPdfua(a) == ua {
			return a, ua, nil
		}

		return "", "", &profileError{detail: fmt.Sprintf("pairs a PDF/A profile with the PDF/UA profile of its PDF version: %s; got %s", validPairs(), p)}
	}

	return "", "", &profileError{detail: fmt.Sprintf("must be one of %s, PDF/UA-1, PDF/UA-2 or %s, or a pair of a PDF/A profile and a PDF/UA one: %s; got %s", strings.Join(sortedKeys(pdfaProfiles), ", "), facturXProfile, validPairs(), p)}
}

// profileError refuses a profile, or a pair of profiles. Its detail follows
// the name of the option or the form field that sent them.
type profileError struct {
	detail string
}

func (e *profileError) Error() string {
	return `"profile" ` + e.detail
}

// pairedPdfua returns the PDF/UA profile the PDF/A profile, or Factur-X,
// pairs with, "" for any other profile. PDF/UA-1 requires a PDF version from
// 1.0 to 1.7 (ISO 14289-1, 6.1), the versions PDF/A-1 to PDF/A-3 build on,
// Factur-X being PDF/A-3. PDF/UA-2 requires PDF 2.0 (ISO 14289-2, 6.1), and
// PDF/A-4 a PDF 2.x version (ISO 19005-4, 6.1.2).
func pairedPdfua(profile string) string {
	switch {
	case profile == facturXProfile:
		return "PDF/UA-1"
	case pdfaProfiles[profile] == "":
		return ""
	case strings.HasPrefix(profile, "PDF/A-4"):
		return "PDF/UA-2"
	default:
		return "PDF/UA-1"
	}
}

// validPairs lists, for a client, the PDF/A profiles, and Factur-X, each
// PDF/UA profile pairs with.
func validPairs() string {
	var withUa1, withUa2 []string
	for _, profile := range sortedKeys(pdfaProfiles) {
		if pairedPdfua(profile) == "PDF/UA-2" {
			withUa2 = append(withUa2, profile)
		} else {
			withUa1 = append(withUa1, profile)
		}
	}

	withUa1 = append(withUa1, facturXProfile)

	return fmt.Sprintf("%s with PDF/UA-1, and %s with PDF/UA-2", joinOr(withUa1), joinOr(withUa2))
}

// joinAnd joins items as "a, b and c".
func joinAnd(items []string) string {
	if len(items) < 2 {
		return strings.Join(items, "")
	}

	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}

// joinOr joins items as "a, b or c".
func joinOr(items []string) string {
	if len(items) < 2 {
		return strings.Join(items, "")
	}

	return strings.Join(items[:len(items)-1], ", ") + " or " + items[len(items)-1]
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
	Dpi     *int   `json:"dpi"`
	Format  string `json:"format"`
	Quality *int   `json:"quality"`
}

func (*renderStage) op() string { return "render" }

func (s *renderStage) validate() error {
	if s.Dpi != nil && *s.Dpi < 1 {
		return fmt.Errorf(`"dpi" must be at least 1, got %d`, *s.Dpi)
	}

	switch s.Format {
	case "", "png":
		if s.Quality != nil {
			return errors.New(`"quality" only applies to the jpeg format`)
		}
	case "jpeg":
		if s.Quality != nil && (*s.Quality < 1 || *s.Quality > 100) {
			return fmt.Errorf(`"quality" must be between 1 and 100, got %d`, *s.Quality)
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
	_ stage = (*composeStage)(nil)
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
	_ stage = (*extractAttachmentsStage)(nil)
	_ stage = (*extractTextStage)(nil)
)
