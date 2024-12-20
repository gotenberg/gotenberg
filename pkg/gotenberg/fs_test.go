package gotenberg

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestOsMkdirAll_MkdirAll(t *testing.T) {
	dirPath, err := NewFileSystem(new(OsMkdirAll)).MkdirAll()
	if err != nil {
		t.Fatalf("create working directory: %v", err)
	}

	err = os.RemoveAll(dirPath)
	if err != nil {
		t.Fatalf("remove working directory: %v", err)
	}
}

func TestOsPathRename_Rename(t *testing.T) {
	dirPath, err := NewFileSystem(new(OsMkdirAll)).MkdirAll()
	if err != nil {
		t.Fatalf("create working directory: %v", err)
	}

	path := "/tests/test/testdata/api/sample1.txt"
	copyPath := filepath.Join(dirPath, fmt.Sprintf("%s.txt", uuid.NewString()))

	in, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}

	defer func() {
		err := in.Close()
		if err != nil {
			t.Fatalf("close file: %v", err)
		}
	}()

	out, err := os.Create(copyPath)
	if err != nil {
		t.Fatalf("create new file: %v", err)
	}

	defer func() {
		err := out.Close()
		if err != nil {
			t.Fatalf("close new file: %v", err)
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		t.Fatalf("copy file to new file: %v", err)
	}

	rename := new(OsPathRename)
	newPath := filepath.Join(dirPath, fmt.Sprintf("%s.txt", uuid.NewString()))

	err = rename.Rename(copyPath, newPath)
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	err = os.RemoveAll(dirPath)
	if err != nil {
		t.Fatalf("remove working directory: %v", err)
	}
}

func TestFileSystem_WorkingDir(t *testing.T) {
	fs := NewFileSystem(new(MkdirAllMock))
	dirName := fs.WorkingDir()

	if dirName == "" {
		t.Error("expected directory name but got empty string")
	}
}

func TestFileSystem_WorkingDirPath(t *testing.T) {
	fs := NewFileSystem(new(MkdirAllMock))
	expectedPath := fmt.Sprintf("%s/%s", os.TempDir(), fs.WorkingDir())

	if fs.WorkingDirPath() != expectedPath {
		t.Errorf("expected path '%s' but got '%s'", expectedPath, fs.WorkingDirPath())
	}
}

func TestFileSystem_NewDirPath(t *testing.T) {
	fs := NewFileSystem(new(MkdirAllMock))
	newDir := fs.NewDirPath()
	expectedPrefix := fs.WorkingDirPath()

	if !strings.HasPrefix(newDir, expectedPrefix) {
		t.Errorf("expected new directory to start with '%s' but got '%s'", expectedPrefix, newDir)
	}
}

func TestFileSystem_MkdirAll(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		mkdirAll    MkdirAll
		expectError bool
	}{
		{
			scenario: "error",
			mkdirAll: &MkdirAllMock{
				MkdirAllMock: func(path string, perm os.FileMode) error {
					return errors.New("foo")
				},
			},
			expectError: true,
		},
		{
			scenario: "success",
			mkdirAll: &MkdirAllMock{
				MkdirAllMock: func(path string, perm os.FileMode) error {
					return nil
				},
			},
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			fs := NewFileSystem(tc.mkdirAll)

			_, err := fs.MkdirAll()

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestWalkDir(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		dir         string
		ext         string
		expectError bool
		expectFiles []string
	}{
		{
			scenario:    "directory does not exist",
			dir:         uuid.NewString(),
			ext:         ".pdf",
			expectError: true,
		},
		{
			scenario: "find PDF files",
			dir: func() string {
				path := fmt.Sprintf("%s/a_directory", os.TempDir())

				err := os.MkdirAll(path, 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/a_foo_file.pdf", path), []byte{1}, 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				err = os.WriteFile(fmt.Sprintf("%s/a_bar_file.PDF", path), []byte{1}, 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				err = os.WriteFile(fmt.Sprintf("%s/a_baz_file.txt", path), []byte{1}, 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return path
			}(),
			ext:         ".pdf",
			expectError: false,
			expectFiles: []string{"/tmp/a_directory/a_bar_file.PDF", "/tmp/a_directory/a_foo_file.pdf"},
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			defer func() {
				err := os.RemoveAll(tc.dir)
				if err != nil {
					t.Fatalf("expected no error while cleaning up but got: %v", err)
				}
			}()

			files, err := WalkDir(tc.dir, tc.ext)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if tc.expectError && err != nil {
				return
			}

			if !reflect.DeepEqual(files, tc.expectFiles) {
				t.Errorf("expected files %+v, but got %+v", tc.expectFiles, files)
			}
		})
	}
}
