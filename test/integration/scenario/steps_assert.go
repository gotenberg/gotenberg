package scenario

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/cucumber/godog"
)

func (s *scenario) allProbeResponseStatusCodesShouldBe(expected int) error {
	if len(s.probeResps) == 0 {
		return errors.New("no probe responses recorded")
	}

	for i, resp := range s.probeResps {
		if resp.Code != expected {
			return fmt.Errorf("probe %d: expected status %d, got %d %q", i+1, expected, resp.Code, resp.Body.String())
		}
	}

	return nil
}

func (s *scenario) allSequentialResponseStatusCodesShouldBe(expected int) error {
	if len(s.sequentialResps) == 0 {
		return errors.New("no sequential responses recorded")
	}

	for i, resp := range s.sequentialResps {
		if resp.Code != expected {
			return fmt.Errorf("sequential response %d: expected status %d, got %d %q", i+1, expected, resp.Code, resp.Body.String())
		}
	}

	return nil
}

func (s *scenario) someConcurrentResponseStatusCodeShouldBe(expected int) error {
	if len(s.concurrentResps) == 0 {
		return errors.New("no concurrent responses recorded")
	}

	codes := make([]int, len(s.concurrentResps))
	for i, resp := range s.concurrentResps {
		if resp.Code == expected {
			return nil
		}
		codes[i] = resp.Code
	}

	return fmt.Errorf("expected at least one concurrent response with status %d, got %v", expected, codes)
}

func (s *scenario) theRequestMethodShouldBe(kind, expected string) error {
	rec, err := s.received(kind)
	if err != nil {
		return err
	}

	if rec.method != expected {
		return fmt.Errorf("expected %s method to be %q, but actual is %q", kind, expected, rec.method)
	}

	return nil
}

func (s *scenario) allConcurrentResponseStatusCodesShouldBe(expected int) error {
	if len(s.concurrentResps) == 0 {
		return errors.New("no concurrent responses recorded")
	}

	for i, resp := range s.concurrentResps {
		if resp.Code != expected {
			return fmt.Errorf("concurrent response %d: expected status %d, got %d %q", i+1, expected, resp.Code, resp.Body.String())
		}
	}

	return nil
}

func (s *scenario) allConcurrentResponsesShouldHavePdfs(expected int) error {
	if len(s.concurrentResps) == 0 {
		return errors.New("no concurrent responses recorded")
	}

	for i, resp := range s.concurrentResps {
		traceID := resp.Header().Get("Gotenberg-Trace")
		dirPath := fmt.Sprintf("%s/%s", s.workdir, traceID)

		_, err := os.Stat(dirPath)
		if os.IsNotExist(err) {
			return fmt.Errorf("concurrent response %d: directory %q does not exist", i+1, dirPath)
		}

		var paths []string
		err = filepath.Walk(dirPath, func(path string, info os.FileInfo, pathErr error) error {
			if pathErr != nil {
				return pathErr
			}
			if strings.EqualFold(filepath.Ext(info.Name()), ".pdf") {
				paths = append(paths, path)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("concurrent response %d: walk %q: %w", i+1, dirPath, err)
		}

		if len(paths) != expected {
			return fmt.Errorf("concurrent response %d: expected %d PDF(s), got %d", i+1, expected, len(paths))
		}
	}

	return nil
}

// received returns the last request of the given kind the helper server got.
func (s *scenario) received(kind string) (*request, error) {
	if s.server == nil {
		return nil, errors.New("server not initialized")
	}

	return s.server.received(kind)
}

func (s *scenario) theGotenbergContainerShouldLogTheFollowingEntries(ctx context.Context, should string, entriesTable *godog.Table) error {
	if s.gotenbergContainer == nil {
		return errors.New("no Gotenberg container")
	}

	expected := make([]string, len(entriesTable.Rows))
	for i, row := range entriesTable.Rows {
		expected[i] = row.Cells[0].Value
	}

	invert := should == "should NOT"
	check := func() error {
		logs, err := containerLogEntries(ctx, s.gotenbergContainer)
		if err != nil {
			return fmt.Errorf("get log entries: %w", err)
		}

		for _, entry := range expected {
			if !invert && !strings.Contains(logs, entry) {
				return fmt.Errorf("expected log entry %q not found in %q", expected, logs)
			}

			if invert && strings.Contains(logs, entry) {
				return fmt.Errorf("log entry %q NOT expected", expected)
			}
		}

		return nil
	}

	var err error
	for range 3 {
		err = check()
		if err != nil && !invert {
			// We have to retry as not all logs may have been produced.
			time.Sleep(500 * time.Millisecond)
			continue
		}
		break
	}
	return err
}

func (s *scenario) theResponseStatusCodeShouldBe(expected int) error {
	if expected != s.resp.Code {
		return fmt.Errorf("expected response status code to be: %d, but actual is: %d %q", expected, s.resp.Code, s.resp.Body.String())
	}
	return nil
}

func (s *scenario) theHeaderValueShouldBe(kind, name string, expected string) error {
	var actual string
	if kind == "response" {
		actual = s.resp.Header().Get(name)
	} else {
		rec, err := s.received(kind)
		if err != nil {
			return err
		}
		actual = rec.header.Get(name)
	}

	expected = strings.ReplaceAll(expected, "{version}", GotenbergVersion)
	if expected != actual {
		return fmt.Errorf("expected %s header %q to be: %q, but actual is: %q", kind, name, expected, actual)
	}
	return nil
}

func (s *scenario) theWebhookRequestHeaderShouldCarryTraceID(name, expected string) error {
	rec, err := s.received("webhook request")
	if err != nil {
		return err
	}

	value := rec.header.Get(name)
	if value == "" {
		return fmt.Errorf("expected webhook request header %q to be set, but it is empty", name)
	}

	// W3C traceparent format: version "-" trace-id "-" parent-id "-" trace-flags.
	parts := strings.Split(value, "-")
	if len(parts) != 4 {
		return fmt.Errorf("expected webhook request header %q to be a valid traceparent, but got %q", name, value)
	}

	if parts[1] != expected {
		return fmt.Errorf("expected webhook request header %q to carry trace id %q, but actual is %q", name, expected, parts[1])
	}
	return nil
}

func (s *scenario) theCookieValueShouldBe(kind, name, expected string) error {
	var cookies []*http.Cookie
	if kind == "response" {
		cookies = s.resp.Result().Cookies()
	} else {
		rec, err := s.received(kind)
		if err != nil {
			return err
		}
		cookies = rec.cookies
	}

	var actual *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == name {
			actual = cookie
			break
		}
	}

	if actual == nil {
		if expected != "" {
			return fmt.Errorf("expected %s cookie %q not found", kind, name)
		}
		return nil
	}

	if expected != actual.Value {
		return fmt.Errorf("expected %s cookie %q to be: %q, but actual is: %q", kind, name, expected, actual.Value)
	}

	return nil
}

func (s *scenario) theBodyShouldMatchString(kind string, expectedDoc *godog.DocString) error {
	var actual string
	if kind == "response" {
		actual = s.resp.Body.String()
	} else {
		rec, err := s.received(kind)
		if err != nil {
			return err
		}
		actual = string(rec.body)
	}

	expected := strings.ReplaceAll(expectedDoc.Content, "{version}", GotenbergVersion)

	if actual != expected {
		return fmt.Errorf("expected %q body to be: %q, but actual is: %q", kind, expected, actual)
	}
	return nil
}

func (s *scenario) theBodyShouldContainString(kind string, expectedDoc *godog.DocString) error {
	var actual string
	if kind == "response" {
		actual = s.resp.Body.String()
	} else {
		rec, err := s.received(kind)
		if err != nil {
			return err
		}
		actual = string(rec.body)
	}

	expected := strings.ReplaceAll(expectedDoc.Content, "{version}", GotenbergVersion)

	if !strings.Contains(actual, expected) {
		return fmt.Errorf("expected %q body to contain: %q, but actual is: %q", kind, expected, actual)
	}
	return nil
}

// theBodyShouldContainTheFollowingLines checks that every line of the
// docstring appears in the body, in any order, so that an assertion does not
// depend on the order a route validates its fields in.
func (s *scenario) theBodyShouldContainTheFollowingLines(kind string, expectedDoc *godog.DocString) error {
	var actual string
	if kind == "response" {
		actual = s.resp.Body.String()
	} else {
		rec, err := s.received(kind)
		if err != nil {
			return err
		}
		actual = string(rec.body)
	}

	var missing []string
	for line := range strings.SplitSeq(expectedDoc.Content, "\n") {
		line = strings.TrimSpace(strings.ReplaceAll(line, "{version}", GotenbergVersion))
		if line != "" && !strings.Contains(actual, line) {
			missing = append(missing, line)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("expected %s body to contain %q, but actual is: %q", kind, missing, actual)
	}
	return nil
}

func (s *scenario) theBodyShouldMatchJSON(kind string, expectedDoc *godog.DocString) error {
	var body []byte
	if kind == "response" {
		body = s.resp.Body.Bytes()
	} else {
		rec, err := s.received(kind)
		if err != nil {
			return err
		}
		body = rec.body
	}

	var expected, actual any

	content := strings.ReplaceAll(expectedDoc.Content, "{version}", GotenbergVersion)
	err := json.Unmarshal([]byte(content), &expected)
	if err != nil {
		return fmt.Errorf("unmarshal expected JSON: %w", err)
	}

	err = json.Unmarshal(body, &actual)
	if err != nil {
		return fmt.Errorf("unmarshal actual JSON: %w", err)
	}

	err = compareJson(expected, actual)
	if err != nil {
		return fmt.Errorf("expected matching JSON: %w", err)
	}

	return nil
}

func (s *scenario) theWebhookEventShouldMatchJSON(ctx context.Context, expectedDoc *godog.DocString) error {
	if s.server == nil {
		return errors.New("server not initialized")
	}

	// Poll briefly — the event fires right after the main webhook.
	var body []byte
	deadline := time.After(5 * time.Second)
	for {
		body = s.server.getEventBody()
		if body != nil {
			break
		}
		select {
		case <-deadline:
			return errors.New("timed out waiting for webhook event")
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}

	var expected, actual any

	content := strings.ReplaceAll(expectedDoc.Content, "{version}", GotenbergVersion)
	err := json.Unmarshal([]byte(content), &expected)
	if err != nil {
		return fmt.Errorf("unmarshal expected JSON: %w", err)
	}

	err = json.Unmarshal(body, &actual)
	if err != nil {
		return fmt.Errorf("unmarshal actual JSON: %w", err)
	}

	err = compareJson(expected, actual)
	if err != nil {
		return fmt.Errorf("expected matching webhook event JSON: %w", err)
	}

	return nil
}

func (s *scenario) thereShouldBePdfs(expected int, kind string) error {
	dirPath := s.teststoreDir

	_, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("directory %q does not exist", dirPath)
	}

	var paths []string
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, pathErr error) error {
		if pathErr != nil {
			return pathErr
		}
		if strings.EqualFold(filepath.Ext(info.Name()), ".pdf") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk %q: %w", s.workdir, err)
	}

	if len(paths) != expected {
		return fmt.Errorf("expected %d PDF(s), but actual is %d", expected, len(paths))
	}

	return nil
}

func (s *scenario) thereShouldBeTheFollowingFiles(kind string, filesTable *godog.Table) error {
	dirPath := s.teststoreDir

	_, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("directory %q does not exist", dirPath)
	}

	var filenames []string
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, pathErr error) error {
		if pathErr != nil {
			return pathErr
		}
		if !info.IsDir() {
			filenames = append(filenames, info.Name())
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk %q: %w", s.workdir, err)
	}

	for _, row := range filesTable.Rows {
		found := false
		expected := row.Cells[0].Value
		for _, filename := range filenames {
			if strings.HasPrefix(expected, "*_") && strings.Contains(filename, strings.ReplaceAll(expected, "*_", "")) {
				found = true
			}
			if strings.EqualFold(expected, filename) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("expected file %q not found among %q", expected, filenames)
		}
	}

	return nil
}

func (s *scenario) theImageShouldBePixels(_ context.Context, name string, width, height int) error {
	path := fmt.Sprintf("%s/%s/%s", s.workdir, s.resp.Header().Get("Gotenberg-Trace"), name)

	file, err := os.Open(path) //nolint:gosec // path is built from test-controlled values.
	if err != nil {
		return fmt.Errorf("open image %q: %w", path, err)
	}
	defer file.Close()

	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return fmt.Errorf("decode image %q: %w", path, err)
	}

	if config.Width != width || config.Height != height {
		return fmt.Errorf("expected %s image %dx%d, but actual is %dx%d", format, width, height, config.Width, config.Height)
	}

	return nil
}

func (s *scenario) theImagePixelShouldBe(_ context.Context, name string, x, y int, want string) error {
	path := fmt.Sprintf("%s/%s/%s", s.workdir, s.resp.Header().Get("Gotenberg-Trace"), name)

	file, err := os.Open(path) //nolint:gosec // path is built from test-controlled values.
	if err != nil {
		return fmt.Errorf("open image %q: %w", path, err)
	}
	defer file.Close()

	img, format, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("decode image %q: %w", path, err)
	}

	r, g, b, _ := img.At(x, y).RGBA()
	// RGBA returns 16-bit channels; shift down to the 8-bit hex form.
	got := fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("expected %s pixel at %d,%d to be %s, but actual is %s", format, x, y, want, got)
	}

	return nil
}

func (s *scenario) theFileSizeShouldBe(_ context.Context, name, comparator string, sizeKB int) error {
	path := fmt.Sprintf("%s/%s/%s", s.workdir, s.resp.Header().Get("Gotenberg-Trace"), name)

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %q: %w", path, err)
	}

	limit := int64(sizeKB) * 1024
	switch comparator {
	case "less":
		if info.Size() >= limit {
			return fmt.Errorf("expected %s (%d bytes) to be smaller than %d KB", name, info.Size(), sizeKB)
		}
	case "greater":
		if info.Size() <= limit {
			return fmt.Errorf("expected %s (%d bytes) to be larger than %d KB", name, info.Size(), sizeKB)
		}
	}

	return nil
}

func (s *scenario) thePdfShouldHaveTheFollowingContentAtPage(ctx context.Context, name, kind string, page int, expected *godog.DocString) error {
	output, err := s.pdfPageText(ctx, name, page)
	if err != nil {
		return err
	}

	invert := kind == "should NOT"

	if !invert && !strings.Contains(output, expected.Content) {
		return fmt.Errorf("expected %q not found in %q", expected.Content, output)
	}

	if invert && strings.Contains(output, expected.Content) {
		return fmt.Errorf("%q found in %q", expected.Content, output)
	}

	return nil
}

func (s *scenario) thePdfShouldHaveContentMatchingAtPage(ctx context.Context, name, kind, pattern string, page int) error {
	output, err := s.pdfPageText(ctx, name, page)
	if err != nil {
		return err
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("compile pattern %q: %w", pattern, err)
	}

	invert := kind == "should NOT"

	if !invert && !re.MatchString(output) {
		return fmt.Errorf("pattern %q not found in %q", pattern, output)
	}

	if invert && re.MatchString(output) {
		return fmt.Errorf("pattern %q found in %q", pattern, output)
	}

	return nil
}
