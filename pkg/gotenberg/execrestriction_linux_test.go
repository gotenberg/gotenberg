//go:build linux

package gotenberg

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

const (
	restrictExecChildEnv = "GOTENBERG_TEST_RESTRICT_EXEC_CHILD"
	// restrictExecUnavailable is the exit code of the child when the kernel
	// or the binary cannot restrict exec.
	restrictExecUnavailable = 3
)

// TestRestrictExec restricts a child copy of the test binary, since the
// restriction cannot be undone, then checks what it may still execute.
func TestRestrictExec(t *testing.T) {
	if os.Getenv(restrictExecChildEnv) == "1" {
		os.Exit(restrictExecChild())
	}

	for _, bin := range []string{"/usr/bin/true", "/usr/bin/false"} {
		_, err := os.Stat(bin)
		if err != nil {
			t.Skipf("%s is missing: %v", bin, err)
		}
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestRestrictExec$", "-test.v") //nolint:gosec // the test binary itself
	cmd.Env = append(os.Environ(), restrictExecChildEnv+"=1", "TMPDIR="+t.TempDir())

	out, err := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == restrictExecUnavailable {
		t.Skipf("exec restriction unavailable here:\n%s", out)
	}

	if err != nil {
		t.Fatalf("restricted child failed: %v\n%s", err, out)
	}

	t.Logf("restricted child:\n%s", out)
}

// restrictExecChild allows /usr/bin/true alone, then reports through its
// exit code whether another binary and a dropped copy are refused. It leaves
// out what Landlock does not cover; see [RestrictExec].
func restrictExecChild() int {
	paths, err := resolveExecPaths([]string{"/usr/bin/true"})
	if err != nil {
		fmt.Printf("resolve paths: %v\n", err)
		return 1
	}

	fmt.Printf("allowed: %v\n", paths)

	err = RestrictExec(paths)
	if errors.Is(err, ErrExecRestrictionUnavailable) {
		fmt.Println(err)
		return restrictExecUnavailable
	}
	if err != nil {
		fmt.Printf("restrict exec: %v\n", err)
		return 1
	}

	err = exec.Command("/usr/bin/true").Run()
	if err != nil {
		fmt.Printf("allowed /usr/bin/true failed: %v\n", err)
		return 1
	}

	copied := filepath.Join(os.TempDir(), "dropped")
	content, err := os.ReadFile("/usr/bin/true")
	if err != nil {
		fmt.Printf("read /usr/bin/true: %v\n", err)
		return 1
	}

	err = os.WriteFile(copied, content, 0o755) //nolint:gosec // G306: must be executable
	if err != nil {
		fmt.Printf("write the copy: %v\n", err)
		return 1
	}

	failed := false
	for _, bin := range []string{
		"/usr/bin/false",
		copied,
	} {
		cmd := exec.Command(bin) //nolint:gosec // fixed test paths
		err = cmd.Run()
		if errors.Is(err, syscall.EACCES) {
			fmt.Printf("refused %s: %v\n", bin, err)
			continue
		}

		fmt.Printf("expected EACCES executing %s but got: %v\n", bin, err)
		failed = true
	}

	if failed {
		return 1
	}

	return 0
}
