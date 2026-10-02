package scenario

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (s *scenario) thePdfShouldReuseImageXObjects(ctx context.Context, name string, images, pages int) error {
	path := filepath.Join(s.workdir, s.resp.Header().Get("Gotenberg-Trace"), name)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("stat PDF %q: %w", path, err)
	}

	cmd := []string{"pdfimages", "-list", filepath.Base(path)}
	output, err := execCommandInIntegrationToolsContainer(ctx, cmd, path)
	if err != nil {
		return fmt.Errorf("exec %q: %w", cmd, err)
	}

	return checkImageXObjectReuse(output, images, pages)
}

func checkImageXObjectReuse(output string, images, pages int) error {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 || !strings.HasPrefix(strings.TrimSpace(lines[0]), "page") {
		return fmt.Errorf("invalid pdfimages -list output: %q", output)
	}

	objects := make(map[[2]int]struct{})
	imagesPerPage := make(map[int]int)
	for _, line := range lines[2:] {
		fields := strings.Fields(line)
		if len(fields) < 12 {
			return fmt.Errorf("invalid pdfimages -list row: %q", line)
		}

		page, err := strconv.Atoi(fields[0])
		if err != nil {
			return fmt.Errorf("parse image page %q: %w", fields[0], err)
		}
		if page < 1 || page > pages {
			return fmt.Errorf("image on page %d outside expected range 1-%d", page, pages)
		}

		// pdfimages lists appearances, including repeated references to the
		// same image. Columns 11 and 12 identify the underlying PDF object.
		var ref [2]int
		for i := range ref {
			ref[i], err = strconv.Atoi(fields[10+i])
			if err != nil {
				return fmt.Errorf("parse image object ID in %q: %w", line, err)
			}
		}
		objects[ref] = struct{}{}
		imagesPerPage[page]++
	}

	for page := 1; page <= pages; page++ {
		if imagesPerPage[page] != images {
			return fmt.Errorf("expected %d image appearance(s) on page %d, but actual is %d", images, page, imagesPerPage[page])
		}
	}
	if len(objects) != images {
		return fmt.Errorf("expected %d unique image XObject(s) shared across %d page(s), but actual is %d", images, pages, len(objects))
	}

	return nil
}
