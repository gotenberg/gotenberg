package gotenberg

import (
	"bufio"
	"bytes"
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// maxInterpreterDepth bounds the chain of interpreters a file may need, such
// as a script whose interpreter is itself a script. It matches the kernel's
// BINPRM_MAX_RECURSION.
const maxInterpreterDepth = 4

// ErrExecRestrictionUnavailable happens when the kernel cannot restrict
// which files the process and its children execute.
var ErrExecRestrictionUnavailable = errors.New("exec restriction unavailable")

// ExecRestrictionPaths returns the files and directories that [RestrictExec]
// should allow: those the [ExecPather] modules of ctx declare, extraPaths,
// and the interpreters the kernel runs to execute them, such as the ELF
// program interpreter (ld.so) or the "#!" line of a script. Every path must
// be absolute and exist. The result is sorted and has no duplicates.
func ExecRestrictionPaths(ctx *Context, extraPaths []string) ([]string, error) {
	mods, err := ctx.Modules(new(ExecPather))
	if err != nil {
		return nil, fmt.Errorf("get exec pathers: %w", err)
	}

	var declared []string
	for _, mod := range mods {
		declared = append(declared, mod.(ExecPather).ExecPaths()...)
	}
	declared = append(declared, extraPaths...)

	return resolveExecPaths(declared)
}

// resolveExecPaths validates paths, then adds the interpreters each file
// needs.
func resolveExecPaths(paths []string) ([]string, error) {
	var resolved []string

	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		if !filepath.IsAbs(path) {
			return nil, fmt.Errorf("exec path %q is not absolute", path)
		}

		path = filepath.Clean(path)

		_, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("stat exec path %q: %w", path, err)
		}

		resolved = append(resolved, path)

		interpreters, err := execInterpreters(path, maxInterpreterDepth)
		if err != nil {
			return nil, fmt.Errorf("find interpreters of %q: %w", path, err)
		}

		resolved = append(resolved, interpreters...)
	}

	slices.Sort(resolved)

	return slices.Compact(resolved), nil
}

// execInterpreters returns the interpreter the kernel runs to execute path,
// and the interpreter's own, depth levels deep. A directory, a static binary
// or a file that is neither ELF nor a script has none.
func execInterpreters(path string, depth int) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}

	if !info.Mode().IsRegular() {
		return nil, nil
	}

	interpreter, err := interpreterOf(path)
	if err != nil {
		return nil, err
	}

	if interpreter == "" {
		return nil, nil
	}

	if depth <= 0 {
		return nil, fmt.Errorf("more than %d nested interpreters", maxInterpreterDepth)
	}

	if !filepath.IsAbs(interpreter) {
		return nil, fmt.Errorf("interpreter %q is not absolute", interpreter)
	}

	nested, err := execInterpreters(interpreter, depth-1)
	if err != nil {
		return nil, fmt.Errorf("find interpreters of %q: %w", interpreter, err)
	}

	return append([]string{filepath.Clean(interpreter)}, nested...), nil
}

// interpreterOf returns the interpreter of an ELF file (its PT_INTERP
// segment) or of a script (its "#!" line), or an empty string.
func interpreterOf(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open: %w", err)
	}
	defer f.Close() //nolint:errcheck // read-only

	// Per execve(2), the kernel reads at most 256 bytes of a "#!" line.
	header := make([]byte, 256)
	n, err := io.ReadFull(f, header)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read header: %w", err)
	}
	header = header[:n]

	if bytes.HasPrefix(header, []byte("#!")) {
		line, _, _ := bufio.NewReader(bytes.NewReader(header[2:])).ReadLine()
		fields := strings.Fields(string(line))
		if len(fields) == 0 {
			return "", errors.New(`empty "#!" line`)
		}

		return fields[0], nil
	}

	if !bytes.HasPrefix(header, []byte(elf.ELFMAG)) {
		return "", nil
	}

	ef, err := elf.NewFile(f)
	if err != nil {
		return "", fmt.Errorf("parse ELF: %w", err)
	}

	for _, prog := range ef.Progs {
		if prog.Type != elf.PT_INTERP {
			continue
		}

		raw, err := io.ReadAll(prog.Open())
		if err != nil {
			return "", fmt.Errorf("read PT_INTERP: %w", err)
		}

		return string(bytes.TrimRight(raw, "\x00")), nil
	}

	return "", nil
}
