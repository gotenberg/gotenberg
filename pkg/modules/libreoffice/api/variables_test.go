package api

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
			err := validateVariableName(tc.name)
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

			outputPath, err := ApplyVariables(inputPath, Variables{Values: map[string]string{"a": "1", "b": "2"}})
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

func TestParseVariables(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		raw         string
		expect      Variables
		expectError string
	}{
		{
			scenario: "empty",
			raw:      "",
		},
		{
			scenario: "values and lists",
			raw:      `{"company":"Smith","items":[{"name":"Router","qty":"2"},{}],"none":[]}`,
			expect: Variables{
				Values: map[string]string{"company": "Smith"},
				Lists: map[string][]map[string]string{
					"items": {{"name": "Router", "qty": "2"}, {}},
					"none":  {},
				},
			},
		},
		{
			scenario:    "not a JSON object",
			raw:         `foo`,
			expectError: `value is not a JSON object, like {"legal_address":"221B Baker Street"}: invalid character 'o' in literal false (expecting 'a')`,
		},
		{
			scenario:    "invalid name",
			raw:         `{"b":"x","a b":"y"}`,
			expectError: "variable name 'a b' is invalid: use letters, digits, '_', '.' or '-', starting with a letter or '_'",
		},
		{
			scenario:    "number value",
			raw:         `{"a":1}`,
			expectError: "variable 'a' is invalid: use a string, or an array of objects with string values to repeat a table row",
		},
		{
			scenario:    "null value",
			raw:         `{"a":null}`,
			expectError: "variable 'a' is invalid: use a string, or an array of objects with string values to repeat a table row",
		},
		{
			scenario:    "list of strings",
			raw:         `{"a":["x"]}`,
			expectError: "variable 'a' is invalid: use a string, or an array of objects with string values to repeat a table row",
		},
		{
			scenario:    "list with a null element",
			raw:         `{"a":[null]}`,
			expectError: "variable 'a' is invalid: use a string, or an array of objects with string values to repeat a table row",
		},
		{
			scenario:    "list with a non-string field",
			raw:         `{"a":[{"x":1}]}`,
			expectError: "variable 'a' is invalid: use a string, or an array of objects with string values to repeat a table row",
		},
		{
			scenario:    "list name with a dot",
			raw:         `{"a.b":[]}`,
			expectError: "list name 'a.b' is invalid: a list name cannot contain '.'",
		},
		{
			scenario:    "field name with a dot",
			raw:         `{"a":[{"x.y":"1"}]}`,
			expectError: "field name 'x.y' of list 'a' is invalid: a field name cannot contain '.'",
		},
		{
			scenario:    "invalid field name",
			raw:         `{"a":[{"x y":"1"}]}`,
			expectError: "list 'a': variable name 'x y' is invalid: use letters, digits, '_', '.' or '-', starting with a letter or '_'",
		},
		{
			scenario:    "value shadowing a list field",
			raw:         `{"a":[],"a.x":"1"}`,
			expectError: "variable 'a.x' is ambiguous: 'a' is a list, and 'a.' refers to its fields",
		},
		{
			scenario:    "too many elements",
			raw:         `{"a":[` + strings.Repeat("{},", maxVariableListRows) + `{}]}`,
			expectError: "list 'a' has 10001 elements: the maximum is 10000",
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			variables, err := ParseVariables(tc.raw)
			if tc.expectError != "" {
				if err == nil || err.Error() != tc.expectError {
					t.Fatalf("expected error\n%s\ngot\n%v", tc.expectError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if !reflect.DeepEqual(variables, tc.expect) {
				t.Errorf("expected %+v, got %+v", tc.expect, variables)
			}
		})
	}
}

func TestExpandPartRows(t *testing.T) {
	cell := func(text string) string {
		return `<w:tc><w:p><w:r><w:t>` + text + `</w:t></w:r></w:p></w:tc>`
	}
	preserved := func(text string) string {
		return `<w:tc><w:p><w:r><w:t xml:space="preserve">` + text + `</w:t></w:r></w:p></w:tc>`
	}
	header := `<w:tr><w:trPr><w:tblHeader/></w:trPr>` + cell("Name") + `</w:tr>`

	lists := map[string][]map[string]string{
		"items": {{"name": "Router", "qty": "2"}, {"name": "Cable & co", "qty": "5"}},
		"none":  {},
		"other": {{"name": "x", "_index": "A"}},
	}

	for _, tc := range []struct {
		scenario      string
		part          string
		expectChanged bool
		expectPart    string
		expectError   error
	}{
		{
			scenario:      "one row per element",
			part:          `<w:tbl>` + header + `<w:tr><w:trPr><w:trHeight w:val="300"/></w:trPr>` + cell("${items._index}") + cell("${items.name}") + cell("${items.qty} pcs") + `</w:tr></w:tbl>`,
			expectChanged: true,
			expectPart: `<w:tbl>` + header +
				`<w:tr><w:trPr><w:trHeight w:val="300"/></w:trPr>` + preserved("1") + preserved("Router") + preserved("2 pcs") + `</w:tr>` +
				`<w:tr><w:trPr><w:trHeight w:val="300"/></w:trPr>` + preserved("2") + preserved("Cable &amp; co") + preserved("5 pcs") + `</w:tr>` +
				`</w:tbl>`,
		},
		{
			scenario:      "empty list removes the row",
			part:          `<w:tbl>` + header + `<w:tr>` + cell("${none.name}") + `</w:tr></w:tbl>`,
			expectChanged: true,
			expectPart:    `<w:tbl>` + header + `</w:tbl>`,
		},
		{
			scenario:      "placeholder split across runs",
			part:          `<w:tbl><w:tr><w:tc><w:p><w:r><w:t>$</w:t></w:r><w:r><w:t>{items.</w:t></w:r><w:r><w:t>name}</w:t></w:r></w:p></w:tc></w:tr></w:tbl>`,
			expectChanged: true,
			expectPart: `<w:tbl>` +
				`<w:tr><w:tc><w:p><w:r><w:t xml:space="preserve">Router</w:t></w:r><w:r><w:t xml:space="preserve"></w:t></w:r><w:r><w:t xml:space="preserve"></w:t></w:r></w:p></w:tc></w:tr>` +
				`<w:tr><w:tc><w:p><w:r><w:t xml:space="preserve">Cable &amp; co</w:t></w:r><w:r><w:t xml:space="preserve"></w:t></w:r><w:r><w:t xml:space="preserve"></w:t></w:r></w:p></w:tc></w:tr>` +
				`</w:tbl>`,
		},
		{
			scenario:      "unknown field and plain variable stay",
			part:          `<w:tbl><w:tr>` + cell("${items.name} ${items.unknown} ${currency}") + `</w:tr></w:tbl>`,
			expectChanged: true,
			expectPart: `<w:tbl>` +
				`<w:tr>` + preserved("Router ${items.unknown} ${currency}") + `</w:tr>` +
				`<w:tr>` + preserved("Cable &amp; co ${items.unknown} ${currency}") + `</w:tr>` +
				`</w:tbl>`,
		},
		{
			scenario:      "element overrides the index",
			part:          `<w:tbl><w:tr>` + cell("${other._index}") + `</w:tr></w:tbl>`,
			expectChanged: true,
			expectPart:    `<w:tbl><w:tr>` + preserved("A") + `</w:tr></w:tbl>`,
		},
		{
			scenario:      "nested row repeats without repeating its parent",
			part:          `<w:tbl><w:tr><w:tc><w:p><w:r><w:t>outer</w:t></w:r></w:p><w:tbl><w:tr>` + cell("${items.name}") + `</w:tr></w:tbl><w:p/></w:tc></w:tr></w:tbl>`,
			expectChanged: true,
			expectPart:    `<w:tbl><w:tr><w:tc><w:p><w:r><w:t>outer</w:t></w:r></w:p><w:tbl><w:tr>` + preserved("Router") + `</w:tr><w:tr>` + preserved("Cable &amp; co") + `</w:tr></w:tbl><w:p/></w:tc></w:tr></w:tbl>`,
		},
		{
			scenario:      "no list placeholder in a row",
			part:          `<w:tbl><w:tr>` + cell("${company} ${unknown.name}") + `</w:tr></w:tbl><w:p><w:r><w:t>${items.name}</w:t></w:r></w:p>`,
			expectChanged: false,
		},
		{
			scenario:    "two lists in one row",
			part:        `<w:tbl><w:tr>` + cell("${items.name}") + cell("${other.name}") + `</w:tr></w:tbl>`,
			expectError: ErrVariablesMixedLists,
		},
		{
			scenario:      "unbalanced rows leave the part untouched",
			part:          `<w:tbl><w:tr>` + cell("${items.name}") + `</w:tbl>`,
			expectChanged: false,
		},
		{
			scenario:      "stray row end leaves the part untouched",
			part:          `<w:tbl></w:tr><w:tr>` + cell("${items.name}") + `</w:tr></w:tbl>`,
			expectChanged: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			out, changed, err := expandPartRows([]byte(tc.part), lists)
			if tc.expectError != nil {
				if !errors.Is(err, tc.expectError) {
					t.Fatalf("expected error %v, got %v", tc.expectError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if changed != tc.expectChanged {
				t.Fatalf("expected changed %t, got %t", tc.expectChanged, changed)
			}
			if changed && string(out) != tc.expectPart {
				t.Errorf("expected\n%s\ngot\n%s", tc.expectPart, out)
			}
		})
	}
}

func TestApplyVariables_Lists(t *testing.T) {
	row := func(text string) string {
		return `<w:tr><w:tc><w:p><w:r><w:t>` + text + `</w:t></w:r></w:p></w:tc></w:tr>`
	}
	document := buildZip(t, map[string]string{
		"word/document.xml": `<w:document><w:body><w:tbl>` + row("${items.name} ${currency}") + `</w:tbl></w:body></w:document>`,
	})

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "in.docx")
	err := os.WriteFile(inputPath, document, 0o600)
	if err != nil {
		t.Fatalf("write input: %v", err)
	}

	outputPath, err := ApplyVariables(inputPath, Variables{
		Values: map[string]string{"currency": "USD"},
		Lists:  map[string][]map[string]string{"items": {{"name": "a"}, {"name": "b"}}},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	preserved := func(text string) string {
		return `<w:tr><w:tc><w:p><w:r><w:t xml:space="preserve">` + text + `</w:t></w:r></w:p></w:tc></w:tr>`
	}
	expect := `<w:document><w:body><w:tbl>` + preserved("a USD") + preserved("b USD") + `</w:tbl></w:body></w:document>`
	got := readZip(t, outputPath)["word/document.xml"]
	if got != expect {
		t.Errorf("expected\n%s\ngot\n%s", expect, got)
	}

	// A list that only repeats rows, without any plain variable replaced.
	outputPath, err = ApplyVariables(inputPath, Variables{
		Lists: map[string][]map[string]string{"items": {}},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expect = `<w:document><w:body><w:tbl></w:tbl></w:body></w:document>`
	got = readZip(t, outputPath)["word/document.xml"]
	if got != expect {
		t.Errorf("expected\n%s\ngot\n%s", expect, got)
	}
}
