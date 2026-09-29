package api

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var (
	// ErrVariablesPasswordProtected happens when variables target a
	// password-protected Word document. The rewrite runs before LibreOffice
	// decrypts the file, so the placeholders are unreachable.
	ErrVariablesPasswordProtected = errors.New("variables cannot be applied to a password-protected document")

	// ErrVariablesInvalidDocument happens when a Word document cannot be read
	// or rewritten while applying variables.
	ErrVariablesInvalidDocument = errors.New("variables cannot be applied to an invalid document")
)

// variablesExtensions lists the Word formats [ApplyVariables] supports. They
// all store their text in WordprocessingML parts under word/.
var variablesExtensions = []string{".docx", ".docm", ".dotx", ".dotm"}

// variableName matches a valid variable name, as used in ${name}.
var variableName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.\-]*$`)

// placeholder matches a ${name} placeholder in the raw, still XML-escaped,
// text of a paragraph. '$', '{', '}' and the name characters are never
// entity-encoded by Word, so matching on raw bytes keeps offsets exact.
var placeholder = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_.\-]*)\}`)

// textNode matches a WordprocessingML text element and captures its content.
// A self-closing <w:t/> never matches, as its next tag is not </w:t>.
var textNode = regexp.MustCompile(`<w:t(?:\s[^>]*)?>([^<]*)</w:t>`)

// textBoundary matches markup that separates two text nodes visually: a
// paragraph start or end, a break or a tab. A placeholder never spans one.
var textBoundary = regexp.MustCompile(`<w:p[\s/>]|</w:p>|<w:(?:br|cr|tab)[\s/>]`)

// wordTextPart matches the parts of a Word package that hold user-visible
// text: the body, headers, footers, footnotes, endnotes and comments.
var wordTextPart = regexp.MustCompile(`^word/(?:document|header\d*|footer\d*|footnotes|endnotes|comments)\.xml$`)

// SupportsVariables reports whether [ApplyVariables] can process the file at
// path, based on its extension.
func SupportsVariables(path string) bool {
	return slices.Contains(variablesExtensions, strings.ToLower(filepath.Ext(path)))
}

// VariablesExtensions returns the extensions [SupportsVariables] accepts.
func VariablesExtensions() []string {
	return slices.Clone(variablesExtensions)
}

// ValidateVariableName returns an error if name cannot appear in a ${name}
// placeholder.
func ValidateVariableName(name string) error {
	if !variableName.MatchString(name) {
		return fmt.Errorf("variable name '%s' is invalid: use letters, digits, '_', '.' or '-', starting with a letter or '_'", name)
	}
	return nil
}

// ApplyVariables returns the path to a copy of the Word document at
// inputPath, where each ${name} placeholder whose name is a key of variables
// is replaced by its value. Placeholders without a matching key stay as-is.
// The copy lives next to inputPath, in the request working directory.
//
// Word often splits a placeholder across several runs, for instance after a
// spell check or a partial formatting change. The replacement therefore works
// on the joined text of each paragraph, and writes the value into the run
// where the placeholder starts, which keeps that run's formatting. A '\n' in a
// value becomes a line break and a '\t' becomes a tab.
//
// It returns [ErrVariablesPasswordProtected] for an encrypted document and
// [ErrVariablesInvalidDocument] when the document cannot be rewritten.
func ApplyVariables(inputPath string, variables map[string]string) (string, error) {
	// Resolve the extension to a literal so the new filename is never derived
	// from the (user-controlled) upload name.
	idx := slices.Index(variablesExtensions, strings.ToLower(filepath.Ext(inputPath)))
	if idx < 0 {
		return "", fmt.Errorf("%w: unsupported extension", ErrVariablesInvalidDocument)
	}
	ext := variablesExtensions[idx]

	if DetectPasswordProtection(inputPath) == PasswordProtectionRequired {
		return "", ErrVariablesPasswordProtected
	}

	src, err := os.ReadFile(inputPath)
	if err != nil {
		return "", fmt.Errorf("read document: %w", err)
	}

	out, err := replaceDocumentVariables(src, variables)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrVariablesInvalidDocument, err)
	}

	// A rewrite that dropped, renamed or corrupted an entry must never reach
	// LibreOffice.
	err = validateWorkbook(src, out)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrVariablesInvalidDocument, err)
	}

	dst, err := os.CreateTemp(filepath.Dir(inputPath), "variables-*"+ext)
	if err != nil {
		return "", fmt.Errorf("create document: %w", err)
	}
	defer dst.Close()

	_, err = dst.Write(out)
	if err != nil {
		_ = os.Remove(dst.Name())
		return "", fmt.Errorf("write document: %w", err)
	}

	return dst.Name(), nil
}

// replaceDocumentVariables rewrites the text parts of a Word package. Every
// other entry, and every text part without a replaced placeholder, is copied
// byte-for-byte without recompression.
func replaceDocumentVariables(src []byte, variables map[string]string) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(src), int64(len(src)))
	if err != nil {
		return nil, fmt.Errorf("open document: %w", err)
	}

	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)

	for _, file := range reader.File {
		if !wordTextPart.MatchString(file.Name) {
			err = copyZipEntry(writer, file)
			if err != nil {
				return nil, err
			}
			continue
		}

		data, err := readZipEntry(file)
		if err != nil {
			return nil, err
		}

		rewritten, changed := replacePartVariables(data, variables)
		if !changed {
			err = copyZipEntry(writer, file)
			if err != nil {
				return nil, err
			}
			continue
		}

		header := file.FileHeader
		header.Method = zip.Deflate
		w, err := writer.CreateHeader(&header)
		if err != nil {
			return nil, fmt.Errorf("write part %q: %w", file.Name, err)
		}
		_, err = w.Write(rewritten)
		if err != nil {
			return nil, fmt.Errorf("write part %q: %w", file.Name, err)
		}
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("finalize document: %w", err)
	}

	return buf.Bytes(), nil
}

// readZipEntry decompresses file, bounded by maxDecompressedWorksheet so a
// decompression bomb cannot exhaust memory.
func readZipEntry(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open part %q: %w", file.Name, err)
	}
	defer rc.Close()

	data, err := io.ReadAll(io.LimitReader(rc, maxDecompressedWorksheet+1))
	if err != nil {
		return nil, fmt.Errorf("read part %q: %w", file.Name, err)
	}
	if len(data) > maxDecompressedWorksheet {
		return nil, fmt.Errorf("part %q exceeds %d bytes", file.Name, maxDecompressedWorksheet)
	}

	return data, nil
}

// replacement is a placeholder to replace, located by its offsets in the
// joined text of a paragraph.
type replacement struct {
	start, end int
	value      string
}

// replacePartVariables replaces the placeholders of a WordprocessingML part
// and reports whether anything changed.
func replacePartVariables(data []byte, variables map[string]string) ([]byte, bool) {
	// A split placeholder may have its '$' and '{' in different nodes, so
	// only the brace is a reliable quick check.
	if !bytes.Contains(data, []byte("{")) {
		return nil, false
	}

	nodes := textNode.FindAllSubmatchIndex(data, -1)
	if len(nodes) == 0 {
		return nil, false
	}

	// Maps a node index to its new content, for the nodes that changed.
	contents := make(map[int]string)

	groupStart := 0
	for i := 1; i <= len(nodes); i++ {
		if i < len(nodes) && !textBoundary.Match(data[nodes[i-1][1]:nodes[i][0]]) {
			continue
		}
		replaceGroupVariables(data, nodes[groupStart:i], groupStart, variables, contents)
		groupStart = i
	}

	if len(contents) == 0 {
		return nil, false
	}

	var out bytes.Buffer
	out.Grow(len(data))
	cursor := 0
	for i, node := range nodes {
		content, ok := contents[i]
		if !ok {
			continue
		}
		out.Write(data[cursor:node[0]])
		// xml:space="preserve" keeps the spaces a value or a truncated run
		// may start or end with.
		out.WriteString(`<w:t xml:space="preserve">`)
		out.WriteString(content)
		out.WriteString(`</w:t>`)
		cursor = node[1]
	}
	out.Write(data[cursor:])

	return out.Bytes(), true
}

// replaceGroupVariables replaces the placeholders found in the joined text of
// nodes, a run of text nodes without a boundary between them. It stores the
// new content of each changed node in contents, keyed by offset plus the
// node's index within the group.
func replaceGroupVariables(data []byte, nodes [][]int, offset int, variables map[string]string, contents map[int]string) {
	var joined strings.Builder
	bounds := make([][2]int, len(nodes))
	for i, node := range nodes {
		bounds[i][0] = joined.Len()
		joined.Write(data[node[2]:node[3]])
		bounds[i][1] = joined.Len()
	}

	text := joined.String()
	var replacements []replacement
	for _, match := range placeholder.FindAllStringSubmatchIndex(text, -1) {
		value, ok := variables[text[match[2]:match[3]]]
		if !ok {
			continue
		}
		replacements = append(replacements, replacement{start: match[0], end: match[1], value: renderValue(value)})
	}
	if len(replacements) == 0 {
		return
	}

	for i, bound := range bounds {
		start, end := bound[0], bound[1]
		var b strings.Builder
		cursor := start
		touched := false

		for _, r := range replacements {
			if r.end <= start || r.start >= end {
				continue
			}
			touched = true
			// A placeholder is written in full in the node where it starts;
			// the nodes it continues into only lose its characters.
			if r.start >= start {
				b.WriteString(text[cursor:r.start])
				b.WriteString(r.value)
			}
			cursor = min(r.end, end)
		}

		if !touched {
			continue
		}
		b.WriteString(text[cursor:end])
		contents[offset+i] = b.String()
	}
}

// renderValue escapes value for a <w:t> element. Line breaks and tabs close
// the current text element, insert the matching run content, and open a new
// text element, all within the same run so the formatting carries over.
func renderValue(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")

	var b strings.Builder
	segment := func(s string) {
		// EscapeText never fails on a strings.Builder; it also replaces
		// characters that are invalid in XML.
		_ = xml.EscapeText(&b, []byte(s))
	}

	lines := strings.Split(value, "\n")
	for i, line := range lines {
		if i > 0 {
			b.WriteString(`</w:t><w:br/><w:t xml:space="preserve">`)
		}
		cells := strings.Split(line, "\t")
		for j, cell := range cells {
			if j > 0 {
				b.WriteString(`</w:t><w:tab/><w:t xml:space="preserve">`)
			}
			segment(cell)
		}
	}

	return b.String()
}
