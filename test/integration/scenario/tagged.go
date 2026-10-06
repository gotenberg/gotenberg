package scenario

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (s *scenario) thePdfShouldBeTagged(ctx context.Context, name, kind string) error {
	path := filepath.Join(s.workdir, s.resp.Header().Get("Gotenberg-Trace"), name)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("stat PDF %q: %w", path, err)
	}

	cmd := []string{"pdfinfo", filepath.Base(path)}
	output, err := execCommandInIntegrationToolsContainer(ctx, cmd, path)
	if err != nil {
		return fmt.Errorf("exec %q: %w", cmd, err)
	}

	return checkTagged(output, kind != "should NOT")
}

// checkTagged reads the "Tagged" line of pdfinfo, which reflects the Marked
// entry of the document catalog's MarkInfo dictionary.
func checkTagged(output string, want bool) error {
	for line := range strings.SplitSeq(output, "\n") {
		value, found := strings.CutPrefix(strings.TrimSpace(line), "Tagged:")
		if !found {
			continue
		}

		tagged := strings.TrimSpace(value) == "yes"
		if tagged != want {
			return fmt.Errorf("expected tagged PDF to be %t, but actual is %t", want, tagged)
		}

		return nil
	}

	return fmt.Errorf("invalid pdfinfo output, no Tagged line: %q", output)
}
