package api

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// PasswordProtection describes whether a document requires a password to open.
type PasswordProtection int

const (
	// PasswordProtectionUnknown means the document's encryption state could not
	// be determined.
	PasswordProtectionUnknown PasswordProtection = iota

	// PasswordProtectionNone means the document opens without a password.
	PasswordProtectionNone

	// PasswordProtectionRequired means the document is encrypted.
	PasswordProtectionRequired
)

var (
	// Compound File Binary magic. An encrypted OOXML document is an
	// MS-OFFCRYPTO container, which is a compound file. Per MS-CFB 2.2, the
	// header signature is fixed.
	ole2Magic = []byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1}

	// Local file header signature. Per APPNOTE.TXT 4.3.7, every ZIP entry
	// starts with it, so an intact package starts with it too.
	zipMagic = []byte{0x50, 0x4b, 0x03, 0x04}

	// An unencrypted OOXML document is always a ZIP package, so any of these
	// extensions over a compound file means the payload is encrypted. Legacy
	// binary formats (.doc, .xls, .ppt) are compound files either way and are
	// deliberately absent.
	ooxmlExtensions = map[string]struct{}{
		".docx": {}, ".docm": {}, ".dotx": {}, ".dotm": {},
		".xlsx": {}, ".xlsm": {}, ".xltx": {}, ".xltm": {},
		".pptx": {}, ".pptm": {}, ".potx": {}, ".potm": {},
		".ppsx": {}, ".ppsm": {},
	}
)

// odfManifestSizeLimit caps how much of an ODF manifest is read. The manifest
// is a few kilobytes in practice; the cap stops a crafted archive from
// exhausting memory through its decompressed size.
const odfManifestSizeLimit = 1 << 20

// DetectPasswordProtection reports whether the document at path is encrypted.
//
// Detection is advisory and never fails: an unreadable file, an unknown format
// or a malformed archive all yield [PasswordProtectionUnknown]. It exists to
// refine the diagnosis of a conversion that already failed, since LibreOffice's
// exit codes do not distinguish a missing password from a crash.
func DetectPasswordProtection(path string) PasswordProtection {
	f, err := os.Open(path)
	if err != nil {
		return PasswordProtectionUnknown
	}
	defer func() {
		_ = f.Close()
	}()

	magic := make([]byte, 8)
	n, err := io.ReadFull(f, magic)
	if err != nil && n < len(zipMagic) {
		return PasswordProtectionUnknown
	}
	magic = magic[:n]

	switch {
	case bytes.HasPrefix(magic, ole2Magic):
		if _, ok := ooxmlExtensions[strings.ToLower(filepath.Ext(path))]; ok {
			return PasswordProtectionRequired
		}
		// A legacy binary document is a compound file whether or not it is
		// encrypted; its encryption lives in a stream this cannot cheaply read.
		return PasswordProtectionUnknown
	case bytes.HasPrefix(magic, zipMagic):
		return detectZipPasswordProtection(f)
	default:
		// Flat XML (.fodt), RTF, CSV and everything else carry no encryption.
		return PasswordProtectionUnknown
	}
}

// detectZipPasswordProtection inspects a ZIP package. ODF keeps META-INF/manifest.xml
// in cleartext even when encrypted, declaring each encrypted entry. An OOXML
// package has no manifest, and reaching this point already proves it is not an
// MS-OFFCRYPTO container, so it opens without a password.
func detectZipPasswordProtection(f *os.File) PasswordProtection {
	size, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return PasswordProtectionUnknown
	}

	r, err := zip.NewReader(f, size)
	if err != nil {
		return PasswordProtectionUnknown
	}

	manifest, err := r.Open("META-INF/manifest.xml")
	if err != nil {
		// No manifest: an OOXML package, or a ZIP that is not an office
		// document at all. Neither is encrypted.
		return PasswordProtectionNone
	}
	defer func() {
		_ = manifest.Close()
	}()

	content, err := io.ReadAll(io.LimitReader(manifest, odfManifestSizeLimit))
	if err != nil {
		return PasswordProtectionUnknown
	}

	// Per OpenDocument 1.3 part 3, section 4.16, an encrypted entry carries a
	// <manifest:encryption-data> child.
	if bytes.Contains(content, []byte("encryption-data")) {
		return PasswordProtectionRequired
	}

	return PasswordProtectionNone
}
