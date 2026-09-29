package api

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestReplacePartVariables(t *testing.T) {
	for _, tc := range []struct {
		scenario      string
		part          string
		variables     map[string]string
		expectChanged bool
		expectPart    string
	}{
		{
			scenario:      "placeholder in a single node",
			part:          `<w:p><w:r><w:t>Hello ${name}!</w:t></w:r></w:p>`,
			variables:     map[string]string{"name": "World"},
			expectChanged: true,
			expectPart:    `<w:p><w:r><w:t xml:space="preserve">Hello World!</w:t></w:r></w:p>`,
		},
		{
			scenario:      "placeholder split across runs",
			part:          `<w:p><w:r><w:t>A: $</w:t></w:r><w:proofErr w:type="spellStart"/><w:r><w:rPr><w:b/></w:rPr><w:t>{legal_</w:t></w:r><w:r><w:t>address} end</w:t></w:r></w:p>`,
			variables:     map[string]string{"legal_address": "Baker St"},
			expectChanged: true,
			expectPart:    `<w:p><w:r><w:t xml:space="preserve">A: Baker St</w:t></w:r><w:proofErr w:type="spellStart"/><w:r><w:rPr><w:b/></w:rPr><w:t xml:space="preserve"></w:t></w:r><w:r><w:t xml:space="preserve"> end</w:t></w:r></w:p>`,
		},
		{
			scenario:      "several placeholders in one node",
			part:          `<w:p><w:r><w:t>${a}-${b}-${a}</w:t></w:r></w:p>`,
			variables:     map[string]string{"a": "1", "b": "2"},
			expectChanged: true,
			expectPart:    `<w:p><w:r><w:t xml:space="preserve">1-2-1</w:t></w:r></w:p>`,
		},
		{
			scenario:      "unknown placeholder stays",
			part:          `<w:p><w:r><w:t>${a} ${unknown}</w:t></w:r></w:p>`,
			variables:     map[string]string{"a": "1"},
			expectChanged: true,
			expectPart:    `<w:p><w:r><w:t xml:space="preserve">1 ${unknown}</w:t></w:r></w:p>`,
		},
		{
			scenario:      "no matching placeholder",
			part:          `<w:p><w:r><w:t>${unknown}</w:t></w:r></w:p>`,
			variables:     map[string]string{"a": "1"},
			expectChanged: false,
		},
		{
			scenario:      "value is XML-escaped",
			part:          `<w:p><w:r><w:t>${a} &amp; co</w:t></w:r></w:p>`,
			variables:     map[string]string{"a": `<Smith & "Sons">`},
			expectChanged: true,
			expectPart:    `<w:p><w:r><w:t xml:space="preserve">&lt;Smith &amp; &#34;Sons&#34;&gt; &amp; co</w:t></w:r></w:p>`,
		},
		{
			scenario:      "newlines and tabs become breaks and tabs",
			part:          `<w:p><w:r><w:t>${a}</w:t></w:r></w:p>`,
			variables:     map[string]string{"a": "x\r\ny\tz"},
			expectChanged: true,
			expectPart:    `<w:p><w:r><w:t xml:space="preserve">x</w:t><w:br/><w:t xml:space="preserve">y</w:t><w:tab/><w:t xml:space="preserve">z</w:t></w:r></w:p>`,
		},
		{
			scenario:      "placeholder never spans paragraphs",
			part:          `<w:p><w:r><w:t>$</w:t></w:r></w:p><w:p><w:r><w:t>{a}</w:t></w:r></w:p>`,
			variables:     map[string]string{"a": "1"},
			expectChanged: false,
		},
		{
			scenario:      "paragraph properties are not a boundary",
			part:          `<w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:t>$</w:t></w:r><w:r><w:t>{a}</w:t></w:r></w:p>`,
			variables:     map[string]string{"a": "1"},
			expectChanged: true,
			expectPart:    `<w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:t xml:space="preserve">1</w:t></w:r><w:r><w:t xml:space="preserve"></w:t></w:r></w:p>`,
		},
		{
			scenario:      "self-closing text node is ignored",
			part:          `<w:p><w:r><w:t/><w:t xml:space="preserve">${a}</w:t></w:r></w:p>`,
			variables:     map[string]string{"a": "1"},
			expectChanged: true,
			expectPart:    `<w:p><w:r><w:t/><w:t xml:space="preserve">1</w:t></w:r></w:p>`,
		},
		{
			scenario:      "no brace at all",
			part:          `<w:p><w:r><w:t>plain</w:t></w:r></w:p>`,
			variables:     map[string]string{"a": "1"},
			expectChanged: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			out, changed := replacePartVariables([]byte(tc.part), tc.variables)
			if changed != tc.expectChanged {
				t.Fatalf("expected changed %t, got %t", tc.expectChanged, changed)
			}
			if changed && string(out) != tc.expectPart {
				t.Errorf("expected\n%s\ngot\n%s", tc.expectPart, out)
			}
		})
	}
}

func TestValidateVariableName(t *testing.T) {
	for _, tc := range []struct {
		name        string
		expectError bool
	}{
		{name: "legal_address"},
		{name: "_a.b-c9"},
		{name: "", expectError: true},
		{name: "9a", expectError: true},
		{name: "legal address", expectError: true},
		{name: "a}", expectError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateVariableName(tc.name)
			if tc.expectError != (err != nil) {
				t.Errorf("expected error %t, got %v", tc.expectError, err)
			}
		})
	}
}

func TestSupportsVariables(t *testing.T) {
	for _, tc := range []struct {
		path   string
		expect bool
	}{
		{path: "/tmp/a.docx", expect: true},
		{path: "/tmp/a.DOCM", expect: true},
		{path: "/tmp/a.dotx", expect: true},
		{path: "/tmp/a.dotm", expect: true},
		{path: "/tmp/a.doc", expect: false},
		{path: "/tmp/a.odt", expect: false},
		{path: "/tmp/a.xlsx", expect: false},
	} {
		t.Run(tc.path, func(t *testing.T) {
			if got := SupportsVariables(tc.path); got != tc.expect {
				t.Errorf("expected %t, got %t", tc.expect, got)
			}
		})
	}
}

func TestApplyVariables(t *testing.T) {
	documentXml := `<w:document><w:body><w:p><w:r><w:t>${a}</w:t></w:r></w:p></w:body></w:document>`
	headerXml := `<w:hdr><w:p><w:r><w:t>${b}</w:t></w:r></w:p></w:hdr>`
	stylesXml := `<w:styles><w:t>${a}</w:t></w:styles>`

	document := buildZip(t, map[string]string{
		"[Content_Types].xml": "<Types/>",
		"word/document.xml":   documentXml,
		"word/header1.xml":    headerXml,
		"word/styles.xml":     stylesXml,
	})

	for _, tc := range []struct {
		scenario    string
		content     []byte
		expectError error
		expectParts map[string]string
	}{
		{
			scenario: "replaces the text parts only",
			content:  document,
			expectParts: map[string]string{
				"word/document.xml": `<w:document><w:body><w:p><w:r><w:t xml:space="preserve">1</w:t></w:r></w:p></w:body></w:document>`,
				"word/header1.xml":  `<w:hdr><w:p><w:r><w:t xml:space="preserve">2</w:t></w:r></w:p></w:hdr>`,
				"word/styles.xml":   stylesXml,
			},
		},
		{
			scenario:    "not a zip",
			content:     []byte("not a zip"),
			expectError: ErrVariablesInvalidDocument,
		},
		{
			scenario: "password-protected",
			content: append(
				[]byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1},
				bytes.Repeat([]byte{0x00}, 64)...,
			),
			expectError: ErrVariablesPasswordProtected,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			dir := t.TempDir()
			inputPath := filepath.Join(dir, "in.docx")

			err := os.WriteFile(inputPath, tc.content, 0o600)
			if err != nil {
				t.Fatalf("write input: %v", err)
			}

			outputPath, err := ApplyVariables(inputPath, map[string]string{"a": "1", "b": "2"})
			if tc.expectError != nil {
				if !errors.Is(err, tc.expectError) {
					t.Fatalf("expected error %v, got %v", tc.expectError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if filepath.Dir(outputPath) != dir || filepath.Ext(outputPath) != ".docx" {
				t.Errorf("unexpected output path %q", outputPath)
			}

			parts := readZip(t, outputPath)
			for name, expect := range tc.expectParts {
				if parts[name] != expect {
					t.Errorf("part %q: expected\n%s\ngot\n%s", name, expect, parts[name])
				}
			}
		})
	}
}

func buildZip(t *testing.T, entries map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range entries {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("create entry %q: %v", name, err)
		}
		_, err = f.Write([]byte(content))
		if err != nil {
			t.Fatalf("write entry %q: %v", name, err)
		}
	}
	err := w.Close()
	if err != nil {
		t.Fatalf("close zip: %v", err)
	}

	return buf.Bytes()
}

func readZip(t *testing.T, path string) map[string]string {
	t.Helper()

	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer r.Close()

	parts := make(map[string]string)
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open entry %q: %v", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatalf("read entry %q: %v", f.Name, err)
		}
		parts[f.Name] = string(data)
	}

	return parts
}
