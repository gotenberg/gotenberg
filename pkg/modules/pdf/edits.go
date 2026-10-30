package pdf

import (
	"context"
	"fmt"
	"path/filepath"

	litho "github.com/gotenberg/litho/pkg/pdf"
)

func (s *deleteStage) edit(_ context.Context, w *litho.Writer, env *runEnv) error {
	pages, err := env.pages(s.Pages)
	if err != nil {
		return err
	}

	return w.DeletePages(pages...)
}

func (s *insertStage) edit(ctx context.Context, w *litho.Writer, env *runEnv) error {
	if s.At > env.pageCount {
		return fmt.Errorf("%w: \"at\" is %d, after the last page (%d)", litho.ErrInvalidPageNumber, s.At, env.pageCount)
	}

	source, err := env.openFile(ctx, s.Source)
	if err != nil {
		return err
	}

	var sourcePages []int
	if s.SourcePages != "" {
		count, err := source.PageCount(ctx)
		if err != nil {
			return fmt.Errorf("count the pages of %q: %w", s.Source, err)
		}

		sourcePages, err = litho.ParsePageRanges(s.SourcePages, count)
		if err != nil {
			return err
		}
	}

	return w.InsertPages(s.At, source, sourcePages...)
}

func (s *rotateStage) edit(_ context.Context, w *litho.Writer, env *runEnv) error {
	pages, err := env.pages(s.Pages)
	if err != nil {
		return err
	}

	return w.RotatePages(s.Angle, pages...)
}

func (s *stampTextStage) edit(_ context.Context, w *litho.Writer, env *runEnv) error {
	pages, err := env.pages(s.Pages)
	if err != nil {
		return err
	}

	stamp := litho.TextStamp{
		Text:     s.Text,
		X:        s.X,
		Y:        s.Y,
		FontSize: s.FontSize,
		Opacity:  s.Opacity,
		Rotation: s.Rotation,
	}

	if s.Font != "" {
		stamp.Font = stampFonts[s.Font]
	}

	if s.Color != "" {
		stamp.Color, err = parseColor(s.Color)
		if err != nil {
			return err
		}
	}

	return w.StampText(stamp, pages...)
}

func (s *stampImageStage) edit(_ context.Context, w *litho.Writer, env *runEnv) error {
	pages, err := env.pages(s.Pages)
	if err != nil {
		return err
	}

	data, err := env.read(s.Image)
	if err != nil {
		return err
	}

	mimeType, err := imageMimeType(s.Image)
	if err != nil {
		return err
	}

	return w.StampImage(litho.ImageStamp{
		Data:     data,
		MimeType: mimeType,
		X:        s.X,
		Y:        s.Y,
		Width:    s.Width,
		Height:   s.Height,
		Opacity:  s.Opacity,
		Rotation: s.Rotation,
	}, pages...)
}

func (s *stampPdfStage) edit(ctx context.Context, w *litho.Writer, env *runEnv) error {
	pages, err := env.pages(s.Pages)
	if err != nil {
		return err
	}

	source, err := env.openFile(ctx, s.Pdf)
	if err != nil {
		return err
	}

	page := s.Page
	if page == 0 {
		page = 1
	}

	return w.StampPDF(litho.PDFStamp{
		Source:   source,
		Page:     page,
		X:        s.X,
		Y:        s.Y,
		Scale:    s.Scale,
		Opacity:  s.Opacity,
		Rotation: s.Rotation,
	}, pages...)
}

func (s *setMetadataStage) edit(_ context.Context, w *litho.Writer, _ *runEnv) error {
	w.SetMetadata(litho.Metadata{
		Title:        s.Title,
		Author:       s.Author,
		Subject:      s.Subject,
		Keywords:     s.Keywords,
		Creator:      s.Creator,
		Producer:     s.Producer,
		CreationDate: s.CreationDate,
		ModDate:      s.ModDate,
	})

	return nil
}

func (s *setBookmarksStage) edit(_ context.Context, w *litho.Writer, _ *runEnv) error {
	w.SetBookmarks(s.Bookmarks)

	return nil
}

func (s *setFormStage) edit(_ context.Context, w *litho.Writer, _ *runEnv) error {
	for _, name := range sortedKeys(s.Fields) {
		w.SetFormField(name, s.Fields[name])
	}

	return nil
}

func (*flattenFormStage) edit(_ context.Context, w *litho.Writer, _ *runEnv) error {
	w.FlattenForm()

	return nil
}

func (*flattenTransparencyStage) edit(ctx context.Context, w *litho.Writer, _ *runEnv) error {
	_, err := w.FlattenTransparency(ctx)

	return err
}

func (s *convertColorsStage) edit(ctx context.Context, w *litho.Writer, _ *runEnv) error {
	var target litho.ColorTarget

	switch s.Target {
	case "cmyk":
		target = litho.CMYKColorTarget()
	case "rgb":
		target = litho.RGBColorTarget()
	case "gray":
		target = litho.GrayColorTarget()
	default:
		target = litho.OutputIntentColorTarget()
	}

	_, err := w.ConvertColors(ctx, target)

	return err
}

func (s *redactStage) edit(ctx context.Context, w *litho.Writer, _ *runEnv) error {
	regions := make([]litho.RedactRegion, len(s.Regions))
	for i, r := range s.Regions {
		regions[i] = litho.RedactRegion{
			Page: r.Page,
			Rect: [4]float64{r.X, r.Y, r.X + r.Width, r.Y + r.Height},
		}
	}

	_, err := w.Redact(ctx, regions)

	return err
}

func (*autotagStage) edit(_ context.Context, w *litho.Writer, _ *runEnv) error {
	w.AutoTag()

	return nil
}

func (s *attachStage) edit(_ context.Context, w *litho.Writer, env *runEnv) error {
	data, err := env.read(s.File)
	if err != nil {
		return err
	}

	name := s.Name
	if name == "" {
		name = filepath.Base(s.File)
	}

	mimeType := s.MimeType
	if mimeType == "" {
		mimeType = mimeTypeOf(name)
	}

	if s.Relationship == "" {
		w.AddAttachment(name, data, mimeType)

		return nil
	}

	w.AddAssociatedFile(litho.AssociatedFile{
		Name:         name,
		Data:         data,
		MimeType:     mimeType,
		Relationship: s.Relationship,
	})

	return nil
}

func (s *detachStage) edit(_ context.Context, w *litho.Writer, _ *runEnv) error {
	w.RemoveAttachment(s.Name)

	return nil
}

func (*decryptStage) edit(_ context.Context, w *litho.Writer, _ *runEnv) error {
	w.RemoveEncryption()

	return nil
}

func (s *remediateStage) edit(ctx context.Context, w *litho.Writer, env *runEnv) error {
	var opts []litho.RemediateOption
	if env.processor.fonts != nil {
		opts = append(opts, env.processor.fonts)
	}

	if s.Language != "" {
		opts = append(opts, litho.WithLanguage(s.Language))
	}

	if s.Aggressive {
		opts = append(opts, litho.WithAggressiveFixes())
	}

	if s.FigureAltText != "" {
		opts = append(opts, litho.WithFigureAltText(s.FigureAltText))
	}

	var (
		report *litho.RemediationReport
		err    error
	)

	switch {
	case pdfaProfiles[s.Profile] != "":
		report, err = w.RemediatePDFA(ctx, pdfaProfiles[s.Profile], opts...)
	case pdfuaProfiles[s.Profile] != "":
		report, err = w.RemediatePDFUA(ctx, pdfuaProfiles[s.Profile], opts...)
	case s.Xml == "":
		report, err = w.RepairFacturX(ctx, opts...)
	default:
		var xml []byte
		xml, err = env.read(s.Xml)
		if err != nil {
			return err
		}

		// litho picks the embedded file name and relationship the level
		// requires.
		report, err = w.RemediateFacturX(ctx, litho.FacturXConfig{
			InvoiceXML:   xml,
			Level:        facturXLevels[s.Level],
			DocumentType: s.DocumentType,
		}, opts...)
	}

	if err != nil {
		return err
	}

	env.report = report

	return nil
}

func (s *signStage) edit(_ context.Context, w *litho.Writer, env *runEnv) error {
	certificate, err := env.read(s.Certificate)
	if err != nil {
		return err
	}

	var signer litho.Signer
	if s.Key == "" {
		signer, err = litho.ParseSignerPKCS12(certificate, s.Password)
	} else {
		var key []byte
		key, err = env.read(s.Key)
		if err == nil {
			signer, err = litho.ParseSignerPEM(certificate, key)
		}
	}

	if err != nil {
		return err
	}

	var opts []litho.SignOption
	if s.Reason != "" {
		opts = append(opts, litho.WithSignatureReason(s.Reason))
	}

	if s.Location != "" {
		opts = append(opts, litho.WithSignatureLocation(s.Location))
	}

	if s.ContactInfo != "" {
		opts = append(opts, litho.WithSignatureContactInfo(s.ContactInfo))
	}

	if s.Name != "" {
		opts = append(opts, litho.WithSignerName(s.Name))
	}

	if s.Pades {
		opts = append(opts, litho.WithPAdES())
	}

	if s.TimestampUrl != "" {
		opts = append(opts, litho.WithTimestampAuthority(s.TimestampUrl))
	}

	w.Sign(signer, opts...)

	return nil
}

func (s *timestampStage) edit(_ context.Context, w *litho.Writer, _ *runEnv) error {
	w.Timestamp(s.Url)

	return nil
}

func (s *optimizeStage) writeOptions() ([]litho.WriteOption, error) {
	var opts []litho.WriteOption

	if boolOr(s.ObjectStreams, true) {
		opts = append(opts, litho.WithObjectStreams())
	}

	if boolOr(s.Dedup, true) {
		opts = append(opts, litho.WithDedup())
	}

	if boolOr(s.RecompressImages, true) {
		opts = append(opts, litho.WithImageRecompression())
	}

	if boolOr(s.SubsetFonts, true) {
		opts = append(opts, litho.WithFontSubsetting())
	}

	return opts, nil
}

func (s *encryptStage) writeOptions() ([]litho.WriteOption, error) {
	permissions := int32(-1)

	if s.Permissions != nil {
		permissions = litho.Permissions{
			Print:             boolOr(s.Permissions.Print, true),
			PrintHighQuality:  boolOr(s.Permissions.PrintHighQuality, true),
			ModifyContent:     boolOr(s.Permissions.ModifyContent, true),
			ExtractText:       boolOr(s.Permissions.ExtractText, true),
			ModifyAnnotations: boolOr(s.Permissions.ModifyAnnotations, true),
			FillForms:         boolOr(s.Permissions.FillForms, true),
			Assemble:          boolOr(s.Permissions.Assemble, true),
		}.Mask()
	}

	return []litho.WriteOption{litho.WithEncryption(litho.EncryptConfig{
		UserPassword:  s.UserPassword,
		OwnerPassword: s.OwnerPassword,
		Algorithm:     encryptAlgorithms[s.Algorithm],
		Permissions:   permissions,
	})}, nil
}

// Interface guards.
var (
	_ editor        = (*deleteStage)(nil)
	_ editor        = (*insertStage)(nil)
	_ editor        = (*rotateStage)(nil)
	_ editor        = (*stampTextStage)(nil)
	_ editor        = (*stampImageStage)(nil)
	_ editor        = (*stampPdfStage)(nil)
	_ editor        = (*setMetadataStage)(nil)
	_ editor        = (*setBookmarksStage)(nil)
	_ editor        = (*setFormStage)(nil)
	_ editor        = (*flattenFormStage)(nil)
	_ editor        = (*flattenTransparencyStage)(nil)
	_ editor        = (*convertColorsStage)(nil)
	_ editor        = (*redactStage)(nil)
	_ editor        = (*autotagStage)(nil)
	_ editor        = (*attachStage)(nil)
	_ editor        = (*detachStage)(nil)
	_ editor        = (*decryptStage)(nil)
	_ editor        = (*remediateStage)(nil)
	_ editor        = (*signStage)(nil)
	_ editor        = (*timestampStage)(nil)
	_ writeOptioner = (*optimizeStage)(nil)
	_ writeOptioner = (*encryptStage)(nil)
)
