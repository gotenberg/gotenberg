package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
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

	// ErrVariablesMixedLists happens when a single table row uses the fields
	// of more than one list, so the number of rows to produce is ambiguous.
	ErrVariablesMixedLists = errors.New("variables cannot repeat a table row that uses more than one list")

	// ErrVariablesTooLarge happens when repeating table rows makes a part of
	// the document exceed its size bound.
	ErrVariablesTooLarge = errors.New("variables make the document too large")
)

// maxVariableListRows bounds how many elements a list may hold, hence how
// many times a table row may repeat.
const maxVariableListRows = 10000

// listIndexField is the field that holds the 1-based position of an element
// within its list, unless the element defines it.
const listIndexField = "_index"

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

// rowToken matches the start or the end of a table row. <w:trPr> and
// <w:trHeight> do not match.
var rowToken = regexp.MustCompile(`<w:tr[\s>]|</w:tr>`)

// Variables holds the values [ApplyVariables] substitutes.
type Variables struct {
	// Values maps a name to the text that replaces its ${name} placeholder.
	Values map[string]string

	// Lists maps a name to its elements. A table row that uses a
	// ${name.field} placeholder repeats once per element.
	Lists map[string][]map[string]string
}

// Empty reports whether there is nothing to substitute.
func (v Variables) Empty() bool {
	return len(v.Values) == 0 && len(v.Lists) == 0
}

// ParseVariables decodes the JSON object raw into [Variables]. A string
// value is a plain variable. An array of objects with string values is a
// list. The returned errors are meant for the client.
func ParseVariables(raw string) (Variables, error) {
	var variables Variables
	if raw == "" {
		return variables, nil
	}

	var object map[string]json.RawMessage
	err := json.Unmarshal([]byte(raw), &object)
	if err != nil {
		return variables, fmt.Errorf(`value is not a JSON object, like {"legal_address":"221B Baker Street"}: %w`, err)
	}

	// Sorted so that the reported name is deterministic.
	names := make([]string, 0, len(object))
	for name := range object {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		err = validateVariableName(name)
		if err != nil {
			return Variables{}, err
		}

		var value string
		if json.Unmarshal(object[name], &value) == nil && !bytes.Equal(bytes.TrimSpace(object[name]), []byte("null")) {
			if variables.Values == nil {
				variables.Values = make(map[string]string)
			}
			variables.Values[name] = value
			continue
		}

		var list []map[string]string
		if json.Unmarshal(object[name], &list) != nil || list == nil || slices.ContainsFunc(list, func(element map[string]string) bool { return element == nil }) {
			return Variables{}, fmt.Errorf("variable '%s' is invalid: use a string, or an array of objects with string values to repeat a table row", name)
		}
		err = validateVariableList(name, list)
		if err != nil {
			return Variables{}, err
		}
		if variables.Lists == nil {
			variables.Lists = make(map[string][]map[string]string)
		}
		variables.Lists[name] = list
	}

	// ${list.field} must have a single meaning.
	for _, name := range names {
		list, _, ok := strings.Cut(name, ".")
		if !ok {
			continue
		}
		if _, isList := variables.Lists[list]; isList {
			return Variables{}, fmt.Errorf("variable '%s' is ambiguous: '%s' is a list, and '%s.' refers to its fields", name, list, list)
		}
	}

	return variables, nil
}

// validateVariableList checks the name, the size and the field names of a
// list.
func validateVariableList(name string, list []map[string]string) error {
	if strings.Contains(name, ".") {
		return fmt.Errorf("list name '%s' is invalid: a list name cannot contain '.'", name)
	}
	if len(list) > maxVariableListRows {
		return fmt.Errorf("list '%s' has %d elements: the maximum is %d", name, len(list), maxVariableListRows)
	}

	for _, element := range list {
		fields := make([]string, 0, len(element))
		for field := range element {
			fields = append(fields, field)
		}
		slices.Sort(fields)

		for _, field := range fields {
			if strings.Contains(field, ".") {
				return fmt.Errorf("field name '%s' of list '%s' is invalid: a field name cannot contain '.'", field, name)
			}
			err := validateVariableName(field)
			if err != nil {
				return fmt.Errorf("list '%s': %w", name, err)
			}
		}
	}

	return nil
}

// SupportsVariables reports whether [ApplyVariables] can process the file at
// path, based on its extension.
func SupportsVariables(path string) bool {
	return slices.Contains(variablesExtensions, strings.ToLower(filepath.Ext(path)))
}

// VariablesExtensions returns the extensions [SupportsVariables] accepts.
func VariablesExtensions() []string {
	return slices.Clone(variablesExtensions)
}

// validateVariableName returns an error if name cannot appear in a ${name}
// placeholder.
func validateVariableName(name string) error {
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
// A table row that uses a ${list.field} placeholder repeats once per element
// of the list, and disappears when the list is empty. ${list._index} is the
// 1-based position of the element. A row of a nested table repeats on its
// own, independently of the row that contains it.
//
// Word often splits a placeholder across several runs, for instance after a
// spell check or a partial formatting change. The replacement therefore works
// on the joined text of each paragraph, and writes the value into the run
// where the placeholder starts, which keeps that run's formatting. A '\n' in a
// value becomes a line break and a '\t' becomes a tab.
//
// It returns [ErrVariablesPasswordProtected] for an encrypted document,
// [ErrVariablesInvalidDocument] when the document cannot be rewritten,
// [ErrVariablesMixedLists] when a table row uses more than one list, and
// [ErrVariablesTooLarge] when the repeated rows exceed the size bound.
func ApplyVariables(inputPath string, variables Variables) (string, error) {
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
	if errors.Is(err, ErrVariablesMixedLists) || errors.Is(err, ErrVariablesTooLarge) {
		return "", err
	}
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
func replaceDocumentVariables(src []byte, variables Variables) ([]byte, error) {
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

		// Rows repeat first, so that the plain variables of a repeated row
		// are replaced in every copy.
		expanded, changed, err := expandPartRows(data, variables.Lists)
		if err != nil {
			return nil, fmt.Errorf("part %q: %w", file.Name, err)
		}
		if changed {
			data = expanded
		}

		rewritten, replaced := replacePartVariables(data, variables.Values)
		if !replaced {
			rewritten = data
		}
		if !changed && !replaced {
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

// tableRow is a <w:tr> element, located by its offsets in a part, with the
// rows of the tables nested inside it.
type tableRow struct {
	start, end int
	rows       []*tableRow
}

// expandPartRows repeats the table rows of a WordprocessingML part that use
// the fields of a list, and reports whether anything changed. A part whose
// row tags are not balanced is left untouched.
func expandPartRows(data []byte, lists map[string][]map[string]string) ([]byte, bool, error) {
	if len(lists) == 0 || !bytes.Contains(data, []byte("{")) {
		return nil, false, nil
	}

	root := &tableRow{start: 0, end: len(data)}
	stack := []*tableRow{root}
	for _, token := range rowToken.FindAllIndex(data, -1) {
		current := stack[len(stack)-1]
		if data[token[0]+1] != '/' {
			row := &tableRow{start: token[0]}
			current.rows = append(current.rows, row)
			stack = append(stack, row)
			continue
		}
		if len(stack) == 1 {
			return nil, false, nil
		}
		current.end = token[1]
		stack = stack[:len(stack)-1]
	}
	if len(stack) != 1 {
		return nil, false, nil
	}

	changed := false
	// The bound applies to what the part grows to, not to each row.
	size := len(data)
	var render func(row *tableRow, isRow bool) ([]byte, error)
	render = func(row *tableRow, isRow bool) ([]byte, error) {
		// own is the row without its nested rows: a nested row repeats on
		// its own, so its placeholders must not repeat the row around it.
		var body, own bytes.Buffer
		cursor := row.start
		for _, nested := range row.rows {
			body.Write(data[cursor:nested.start])
			own.Write(data[cursor:nested.start])
			rendered, err := render(nested, true)
			if err != nil {
				return nil, err
			}
			body.Write(rendered)
			cursor = nested.end
		}
		body.Write(data[cursor:row.end])
		own.Write(data[cursor:row.end])

		if !isRow {
			return body.Bytes(), nil
		}

		var used []string
		for _, name := range placeholderNames(own.Bytes()) {
			list, _, ok := strings.Cut(name, ".")
			if !ok {
				continue
			}
			if _, isList := lists[list]; isList && !slices.Contains(used, list) {
				used = append(used, list)
			}
		}
		if len(used) == 0 {
			return body.Bytes(), nil
		}
		if len(used) > 1 {
			slices.Sort(used)
			return nil, fmt.Errorf("%w: '%s'", ErrVariablesMixedLists, strings.Join(used, "', '"))
		}

		changed = true
		elements := lists[used[0]]
		size += body.Len() * (len(elements) - 1)
		if size > maxDecompressedWorksheet {
			return nil, fmt.Errorf("%w: list '%s' with %d elements", ErrVariablesTooLarge, used[0], len(elements))
		}

		var out bytes.Buffer
		for i, element := range elements {
			fields := make(map[string]string, len(element)+1)
			fields[used[0]+"."+listIndexField] = strconv.Itoa(i + 1)
			for field, value := range element {
				fields[used[0]+"."+field] = value
			}

			rendered, replaced := replacePartVariables(body.Bytes(), fields)
			if !replaced {
				rendered = body.Bytes()
			}
			out.Write(rendered)
		}

		return out.Bytes(), nil
	}

	out, err := render(root, false)
	if err != nil {
		return nil, false, err
	}
	if !changed {
		return nil, false, nil
	}

	return out, true, nil
}

// placeholderNames returns the names of the placeholders found in a fragment
// of a WordprocessingML part, including the ones split across text nodes.
func placeholderNames(data []byte) []string {
	nodes := textNode.FindAllSubmatchIndex(data, -1)

	var names []string
	eachTextGroup(data, nodes, func(group [][]int, _ int) {
		var joined strings.Builder
		for _, node := range group {
			joined.Write(data[node[2]:node[3]])
		}
		for _, match := range placeholder.FindAllStringSubmatch(joined.String(), -1) {
			names = append(names, match[1])
		}
	})

	return names
}

// eachTextGroup calls fn for each run of text nodes without a boundary
// between them, with the index of the first node of the run.
func eachTextGroup(data []byte, nodes [][]int, fn func(group [][]int, offset int)) {
	if len(nodes) == 0 {
		return
	}

	groupStart := 0
	for i := 1; i <= len(nodes); i++ {
		if i < len(nodes) && !textBoundary.Match(data[nodes[i-1][1]:nodes[i][0]]) {
			continue
		}
		fn(nodes[groupStart:i], groupStart)
		groupStart = i
	}
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

	eachTextGroup(data, nodes, func(group [][]int, offset int) {
		replaceGroupVariables(data, group, offset, variables, contents)
	})

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
