package gotenberg

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// writeELF writes a minimal ELF64 file whose PT_INTERP segment names interp,
// or a static one without PT_INTERP when interp is empty.
func writeELF(t *testing.T, path, interp string) {
	t.Helper()

	var phnum uint16
	if interp != "" {
		phnum = 1
	}

	ehdrSize := uint64(binary.Size(elf.Header64{}))
	phdrSize := uint64(binary.Size(elf.Prog64{}))

	header := elf.Header64{
		Type:      uint16(elf.ET_EXEC),
		Machine:   uint16(elf.EM_X86_64),
		Version:   uint32(elf.EV_CURRENT),
		Phoff:     ehdrSize,
		Ehsize:    uint16(ehdrSize),
		Phentsize: uint16(phdrSize),
		Phnum:     phnum,
	}
	copy(header.Ident[:], elf.ELFMAG)
	header.Ident[elf.EI_CLASS] = byte(elf.ELFCLASS64)
	header.Ident[elf.EI_DATA] = byte(elf.ELFDATA2LSB)
	header.Ident[elf.EI_VERSION] = byte(elf.EV_CURRENT)

	var buf bytes.Buffer
	err := binary.Write(&buf, binary.LittleEndian, header)
	if err != nil {
		t.Fatalf("write ELF header: %v", err)
	}

	if interp != "" {
		size := uint64(len(interp) + 1)
		err = binary.Write(&buf, binary.LittleEndian, elf.Prog64{
			Type:   uint32(elf.PT_INTERP),
			Flags:  uint32(elf.PF_R),
			Off:    ehdrSize + phdrSize,
			Filesz: size,
			Memsz:  size,
			Align:  1,
		})
		if err != nil {
			t.Fatalf("write ELF program header: %v", err)
		}
		buf.WriteString(interp + "\x00")
	}

	err = os.WriteFile(path, buf.Bytes(), 0o755)
	if err != nil {
		t.Fatalf("write ELF file: %v", err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	err := os.WriteFile(path, []byte(content), 0o755)
	if err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

func TestExecRestrictionPaths(t *testing.T) {
	dir := t.TempDir()

	loader := filepath.Join(dir, "ld.so")
	writeELF(t, loader, "")

	dynamic := filepath.Join(dir, "dynamic")
	writeELF(t, dynamic, loader)

	static := filepath.Join(dir, "static")
	writeELF(t, static, "")

	shell := filepath.Join(dir, "sh")
	writeELF(t, shell, loader)

	launcher := filepath.Join(dir, "launcher")
	writeFile(t, launcher, "#! "+shell+" -e\nexec "+dynamic+"\n")

	nestedLauncher := filepath.Join(dir, "nested-launcher")
	writeFile(t, nestedLauncher, "#!"+launcher+"\n")

	data := filepath.Join(dir, "data")
	writeFile(t, data, "not an executable")

	loop := filepath.Join(dir, "loop")
	writeFile(t, loop, "#!"+loop+"\n")

	missingInterpreter := filepath.Join(dir, "missing-interpreter")
	writeFile(t, missingInterpreter, "#!"+filepath.Join(dir, "nope")+"\n")

	relativeInterpreter := filepath.Join(dir, "relative-interpreter")
	writeFile(t, relativeInterpreter, "#!sh\n")

	emptyShebang := filepath.Join(dir, "empty-shebang")
	writeFile(t, emptyShebang, "#!\n")

	programDir := filepath.Join(dir, "program")
	err := os.Mkdir(programDir, 0o755)
	if err != nil {
		t.Fatalf("create program directory: %v", err)
	}

	for _, tc := range []struct {
		scenario    string
		modPaths    []string
		extraPaths  []string
		expectPaths []string
		expectError bool
	}{
		{
			scenario:    "no paths",
			expectPaths: nil,
		},
		{
			scenario:    "static binary and directory",
			modPaths:    []string{static},
			extraPaths:  []string{programDir + "/"},
			expectPaths: []string{programDir, static},
		},
		{
			scenario:    "dynamic binary adds its program interpreter",
			modPaths:    []string{dynamic},
			expectPaths: []string{dynamic, loader},
		},
		{
			scenario:    "script adds its interpreter and the interpreter's",
			modPaths:    []string{launcher},
			expectPaths: []string{launcher, loader, shell},
		},
		{
			scenario:    "nested scripts",
			modPaths:    []string{nestedLauncher},
			expectPaths: []string{launcher, loader, nestedLauncher, shell},
		},
		{
			scenario:    "duplicates and blank entries",
			modPaths:    []string{dynamic, shell},
			extraPaths:  []string{" ", dynamic},
			expectPaths: []string{dynamic, loader, shell},
		},
		{
			scenario:    "neither ELF nor script",
			extraPaths:  []string{data},
			expectPaths: []string{data},
		},
		{
			scenario:    "relative path",
			extraPaths:  []string{"usr/bin/chromium"},
			expectError: true,
		},
		{
			scenario:    "missing path",
			extraPaths:  []string{filepath.Join(dir, "nope")},
			expectError: true,
		},
		{
			scenario:    "missing interpreter",
			extraPaths:  []string{missingInterpreter},
			expectError: true,
		},
		{
			scenario:    "relative interpreter",
			extraPaths:  []string{relativeInterpreter},
			expectError: true,
		},
		{
			scenario:    "empty shebang",
			extraPaths:  []string{emptyShebang},
			expectError: true,
		},
		{
			scenario:    "interpreter loop",
			extraPaths:  []string{loop},
			expectError: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			mod := &struct {
				ModuleMock
				ExecPatherMock
			}{}
			mod.DescriptorMock = func() ModuleDescriptor {
				return ModuleDescriptor{ID: "foo", New: func() Module { return mod }}
			}
			mod.ExecPathsMock = func() []string { return tc.modPaths }

			ctx := NewContext(ParsedFlags{}, []ModuleDescriptor{mod.Descriptor()})

			paths, err := ExecRestrictionPaths(ctx, tc.extraPaths)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if !reflect.DeepEqual(paths, tc.expectPaths) {
				t.Errorf("expected paths %v but got %v", tc.expectPaths, paths)
			}
		})
	}
}
