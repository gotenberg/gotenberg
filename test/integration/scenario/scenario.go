package scenario

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cucumber/godog"
	"github.com/google/uuid"
	"github.com/mholt/archives"
	"github.com/testcontainers/testcontainers-go"
)

var (
	failedScenariosMu sync.Mutex
	failedScenarios   []string // file:line paths for re-run
)

// ResetFailedScenarios clears the collected failures.
func ResetFailedScenarios() {
	failedScenariosMu.Lock()
	defer failedScenariosMu.Unlock()
	failedScenarios = nil
}

// FailedScenarioPaths returns file:line paths of failed scenarios.
func FailedScenarioPaths() []string {
	failedScenariosMu.Lock()
	defer failedScenariosMu.Unlock()
	result := make([]string, len(failedScenarios))
	copy(result, failedScenarios)
	return result
}

func recordFailedScenario(sc *godog.Scenario) {
	// When godog runs with file:line paths, the Uri may contain
	// the line suffix (e.g., "features/root.feature:4"). Strip it
	// to get the actual file path for reading.
	filePath := sc.Uri
	if idx := strings.LastIndex(filePath, ":"); idx > 0 {
		if _, err := strconv.Atoi(filePath[idx+1:]); err == nil {
			filePath = filePath[:idx]
		}
	}

	line := findScenarioLine(filePath, sc.Name)
	if line <= 0 {
		return
	}

	path := fmt.Sprintf("%s:%d", filePath, line)

	failedScenariosMu.Lock()
	defer failedScenariosMu.Unlock()
	failedScenarios = append(failedScenarios, path)
}

func findScenarioLine(filePath, name string) int {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0
	}

	target := "Scenario: " + name
	for i, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == target {
			return i + 1
		}
	}

	return 0
}

type scenario struct {
	resp                      *httptest.ResponseRecorder
	concurrentResps           []*httptest.ResponseRecorder
	workdir                   string
	teststoreDir              string
	gotenbergContainer        testcontainers.Container
	gotenbergContainerNetwork *testcontainers.DockerNetwork
	server                    *server
	hostPort                  int
}

func (s *scenario) reset(ctx context.Context) error {
	s.resp = httptest.NewRecorder()
	s.concurrentResps = nil

	err := os.RemoveAll(s.workdir)
	if err != nil {
		return fmt.Errorf("remove workdir %q: %w", s.workdir, err)
	}
	s.workdir = ""

	if s.server == nil {
		return nil
	}

	err = s.server.stop(ctx)
	if err != nil {
		return fmt.Errorf("stop server: %w", err)
	}

	return nil
}

func (s *scenario) iHaveADefaultGotenbergContainer(ctx context.Context) error {
	n, c, err := startGotenbergContainer(ctx, nil)
	if err != nil {
		return fmt.Errorf("create Gotenberg container: %s", err)
	}
	s.gotenbergContainerNetwork = n
	s.gotenbergContainer = c
	return nil
}

func (s *scenario) iHaveAGotenbergContainerWithTheFollowingEnvironmentVariables(ctx context.Context, envTable *godog.Table) error {
	env := make(map[string]string)
	for _, row := range envTable.Rows {
		env[row.Cells[0].Value] = row.Cells[1].Value
	}
	n, c, err := startGotenbergContainer(ctx, env)
	if err != nil {
		return fmt.Errorf("create Gotenberg container: %s", err)
	}
	s.gotenbergContainerNetwork = n
	s.gotenbergContainer = c
	return nil
}

func (s *scenario) iHaveAServer(ctx context.Context) error {
	srv, err := newServer(ctx, s.workdir)
	if err != nil {
		return fmt.Errorf("create server: %s", err)
	}
	s.server = srv
	port, err := s.server.start(ctx)
	if err != nil {
		return fmt.Errorf("start server: %s", err)
	}
	s.hostPort = port
	return nil
}

func (s *scenario) iMakeARequestToGotenberg(ctx context.Context, method, endpoint string) error {
	return s.iMakeARequestToGotenbergWithTheFollowingHeaders(ctx, method, endpoint, nil)
}

func (s *scenario) iMakeARequestToGotenbergWithTheFollowingHeaders(ctx context.Context, method, endpoint string, headersTable *godog.Table) error {
	if s.gotenbergContainer == nil {
		return errors.New("no Gotenberg container")
	}

	base, err := containerHttpEndpoint(ctx, s.gotenbergContainer, "3000")
	if err != nil {
		return fmt.Errorf("get container HTTP endpoint: %w", err)
	}

	headers := make(map[string]string)
	if headersTable != nil {
		for _, row := range headersTable.Rows {
			headers[row.Cells[0].Value] = row.Cells[1].Value
		}
	}

	resp, err := doRequest(method, fmt.Sprintf("%s%s", base, endpoint), headers, nil)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	s.resp = httptest.NewRecorder()
	s.resp.Code = resp.StatusCode
	for key, values := range resp.Header {
		for _, v := range values {
			s.resp.Header().Add(key, v)
		}
	}
	_, err = s.resp.Body.Write(body)
	if err != nil {
		return fmt.Errorf("write response body: %w", err)
	}

	return nil
}

func (s *scenario) iMakeARequestToGotenbergWithTheFollowingFormDataAndHeaders(ctx context.Context, method, endpoint string, dataTable *godog.Table) error {
	if s.gotenbergContainer == nil {
		return errors.New("no Gotenberg container")
	}

	fields := make(map[string]string)
	files := make(map[string][]string)
	headers := make(map[string]string)

	for _, row := range dataTable.Rows {
		name := row.Cells[0].Value
		value := row.Cells[1].Value
		kind := row.Cells[2].Value

		switch kind {
		case "field":
			if name == "downloadFrom" || name == "url" || name == "cookies" {
				fields[name] = strings.ReplaceAll(value, "%d", fmt.Sprintf("%d", s.hostPort))
				continue
			}
			fields[name] = value
		case "file":
			if strings.Contains(value, "teststore") {
				if s.teststoreDir == "" {
					return errors.New("no teststore directory available from previous requests")
				}
				_, err := os.Stat(s.teststoreDir)
				if os.IsNotExist(err) {
					return fmt.Errorf("directory %q does not exist", s.teststoreDir)
				}
				value = strings.ReplaceAll(value, "teststore", s.teststoreDir)
			} else {
				wd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("get current directory: %w", err)
				}
				value = fmt.Sprintf("%s/%s", wd, value)
			}
			files[name] = append(files[name], value)
		case "header":
			if name == "Gotenberg-Webhook-Url" || name == "Gotenberg-Webhook-Error-Url" || name == "Gotenberg-Webhook-Events-Url" {
				headers[name] = fmt.Sprintf(value, s.hostPort)
				continue
			}
			headers[name] = value
		default:
			return fmt.Errorf("unexpected %q %q", kind, value)
		}
	}

	base, err := containerHttpEndpoint(ctx, s.gotenbergContainer, "3000")
	if err != nil {
		return fmt.Errorf("get container HTTP endpoint: %w", err)
	}

	resp, err := doFormDataRequest(method, fmt.Sprintf("%s%s", base, endpoint), fields, files, headers)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	s.resp = httptest.NewRecorder()
	s.resp.Code = resp.StatusCode
	for key, values := range resp.Header {
		for _, v := range values {
			s.resp.Header().Add(key, v)
		}
	}
	_, err = s.resp.Body.Write(body)
	if err != nil {
		return fmt.Errorf("write response body: %w", err)
	}

	if resp.StatusCode == http.StatusNoContent {
		// Gotenberg processes this asynchronously. The webhook test server
		// will save the incoming files under this trace ID directory shortly.
		s.teststoreDir = fmt.Sprintf("%s/%s", s.workdir, s.resp.Header().Get("Gotenberg-Trace"))
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	cd := resp.Header.Get("Content-Disposition")
	if cd == "" {
		return nil
	}

	_, params, err := mime.ParseMediaType(cd)
	if err != nil {
		return fmt.Errorf("parse Content-Disposition header: %w", err)
	}

	filename, ok := params["filename"]
	if !ok {
		return errors.New("no filename in Content-Disposition header")
	}

	dirPath := fmt.Sprintf("%s/%s", s.workdir, s.resp.Header().Get("Gotenberg-Trace"))
	err = os.MkdirAll(dirPath, 0o755)
	if err != nil {
		return fmt.Errorf("create working directory: %w", err)
	}

	s.teststoreDir = dirPath

	fpath := fmt.Sprintf("%s/%s", dirPath, filename)
	file, err := os.Create(fpath)
	if err != nil {
		return fmt.Errorf("create file %q: %w", fpath, err)
	}
	defer file.Close()

	_, err = file.Write(body)
	if err != nil {
		return fmt.Errorf("write file %q: %w", fpath, err)
	}

	if resp.Header.Get("Content-Type") == "application/zip" {
		var format archives.Zip
		err = format.Extract(ctx, file, func(ctx context.Context, f archives.FileInfo) error {
			source, err := f.Open()
			if err != nil {
				return fmt.Errorf("open file %q: %w", f.Name(), err)
			}
			defer source.Close()

			targetPath := fmt.Sprintf("%s/%s", dirPath, f.Name())
			target, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("create file %q: %w", targetPath, err)
			}
			defer target.Close()

			_, err = io.Copy(target, source)
			if err != nil {
				return fmt.Errorf("copy file %q: %w", targetPath, err)
			}

			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *scenario) iMakeConcurrentRequestsToGotenberg(ctx context.Context, count int, method, endpoint string, dataTable *godog.Table) error {
	if s.gotenbergContainer == nil {
		return errors.New("no Gotenberg container")
	}

	fields := make(map[string]string)
	files := make(map[string][]string)
	headers := make(map[string]string)

	for _, row := range dataTable.Rows {
		name := row.Cells[0].Value
		value := row.Cells[1].Value
		kind := row.Cells[2].Value

		switch kind {
		case "field":
			fields[name] = value
		case "file":
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get current directory: %w", err)
			}
			value = fmt.Sprintf("%s/%s", wd, value)
			files[name] = append(files[name], value)
		case "header":
			headers[name] = value
		default:
			return fmt.Errorf("unexpected %q %q", kind, value)
		}
	}

	base, err := containerHttpEndpoint(ctx, s.gotenbergContainer, "3000")
	if err != nil {
		return fmt.Errorf("get container HTTP endpoint: %w", err)
	}

	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	s.concurrentResps = make([]*httptest.ResponseRecorder, 0, count)
	errs := make([]error, 0)

	for range count {
		wg.Go(func() {
			resp, reqErr := doFormDataRequest(method, fmt.Sprintf("%s%s", base, endpoint), fields, files, headers)
			if reqErr != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("do request: %w", reqErr))
				mu.Unlock()
				return
			}
			defer resp.Body.Close()

			body, reqErr := io.ReadAll(resp.Body)
			if reqErr != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("read response body: %w", reqErr))
				mu.Unlock()
				return
			}

			rec := httptest.NewRecorder()
			rec.Code = resp.StatusCode
			for key, values := range resp.Header {
				for _, v := range values {
					rec.Header().Add(key, v)
				}
			}
			_, _ = rec.Body.Write(body)

			if resp.StatusCode == http.StatusOK {
				cd := resp.Header.Get("Content-Disposition")
				if cd != "" {
					_, params, parseErr := mime.ParseMediaType(cd)
					if parseErr == nil {
						if filename, ok := params["filename"]; ok {
							traceID := resp.Header.Get("Gotenberg-Trace")
							dirPath := fmt.Sprintf("%s/%s", s.workdir, traceID)

							mu.Lock()
							mkErr := os.MkdirAll(dirPath, 0o755)
							mu.Unlock()

							if mkErr == nil {
								fpath := fmt.Sprintf("%s/%s", dirPath, filename)
								f, fErr := os.Create(fpath)
								if fErr == nil {
									_, _ = f.Write(body)
									f.Close()
								}
							}
						}
					}
				}
			}

			mu.Lock()
			s.concurrentResps = append(s.concurrentResps, rec)
			mu.Unlock()
		})
	}

	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("concurrent requests failed: %v", errs)
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

func (s *scenario) iWaitForTheAsynchronousRequestToWebhook(ctx context.Context) error {
	if s.server == nil {
		return errors.New("server not initialized")
	}
	if s.server.req != nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-s.server.errChan:
		return err
	}
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
	switch {
	case kind == "response":
		actual = s.resp.Header().Get(name)
	case s.server == nil:
		return errors.New("server not initialized")
	case s.server.req == nil:
		return errors.New("no webhook request found")
	default:
		actual = s.server.req.Header.Get(name)
	}

	if expected != actual {
		return fmt.Errorf("expected %s header %q to be: %q, but actual is: %q", kind, name, expected, actual)
	}
	return nil
}

func (s *scenario) theCookieValueShouldBe(kind, name, expected string) error {
	var cookies []*http.Cookie
	switch {
	case kind == "response":
		cookies = s.resp.Result().Cookies()
	case s.server == nil:
		return errors.New("server not initialized")
	case s.server.req == nil:
		return errors.New("no webhook request found")
	default:
		cookies = s.server.req.Cookies()
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
	switch {
	case kind == "response":
		actual = s.resp.Body.String()
	case s.server == nil:
		return errors.New("server not initialized")
	case s.server.req == nil:
		return errors.New("no webhook request found")
	default:
		actual = string(s.server.bodyCopy)
	}

	expected := strings.ReplaceAll(expectedDoc.Content, "{version}", GotenbergVersion)

	if actual != expected {
		return fmt.Errorf("expected %q body to be: %q, but actual is: %q", kind, expected, actual)
	}
	return nil
}

func (s *scenario) theBodyShouldContainString(kind string, expectedDoc *godog.DocString) error {
	var actual string
	switch {
	case kind == "response":
		actual = s.resp.Body.String()
	case s.server == nil:
		return errors.New("server not initialized")
	case s.server.req == nil:
		return errors.New("no webhook request found")
	default:
		actual = string(s.server.bodyCopy)
	}

	expected := strings.ReplaceAll(expectedDoc.Content, "{version}", GotenbergVersion)

	if !strings.Contains(actual, expected) {
		return fmt.Errorf("expected %q body to contain: %q, but actual is: %q", kind, expected, actual)
	}
	return nil
}

func (s *scenario) theBodyShouldMatchJSON(kind string, expectedDoc *godog.DocString) error {
	var body []byte
	switch {
	case kind == "response":
		body = s.resp.Body.Bytes()
	case s.server == nil:
		return errors.New("server not initialized")
	case s.server.req == nil:
		return errors.New("no webhook request found")
	default:
		body = s.server.bodyCopy
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

func (s *scenario) thePdfsShouldBeValidWithAToleranceOf(ctx context.Context, kind, validate string, tolerance int) error {
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

	var flavor string
	switch validate {
	case "PDF/A-1b":
		flavor = "1b"
	case "PDF/A-2b":
		flavor = "2b"
	case "PDF/A-3b":
		flavor = "3b"
	case "PDF/UA-1":
		flavor = "ua1"
	case "PDF/UA-2":
		flavor = "ua2"
	default:
		return fmt.Errorf("unknown %q", validate)
	}

	re := regexp.MustCompile(`failedRules="(\d+)"`)
	for _, path := range paths {
		cmd := []string{
			"verapdf",
			"-f",
			flavor,
			filepath.Base(path),
		}

		output, err := execCommandInIntegrationToolsContainer(ctx, cmd, path)
		if err != nil {
			return fmt.Errorf("exec %q: %w", cmd, err)
		}

		matches := re.FindStringSubmatch(output)
		if len(matches) < 2 {
			return errors.New("expected failed rules")
		}

		failedRules, err := strconv.Atoi(matches[1])
		if err != nil {
			return fmt.Errorf("convert failed rules value %q to integer: %w", matches[1], err)
		}

		if tolerance < failedRules {
			return fmt.Errorf("expected failed rules to be inferior or equal to: %d, but actual is %d", tolerance, failedRules)
		}
	}

	return nil
}

func (s *scenario) thePdfShouldHavePages(ctx context.Context, name string, pages int) error {
	var path string
	if !strings.HasPrefix(name, "*_") {
		path = fmt.Sprintf("%s/%s/%s", s.workdir, s.resp.Header().Get("Gotenberg-Trace"), name)

		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			return fmt.Errorf("PDF %q does not exist", path)
		}
	} else {
		substr := strings.ReplaceAll(name, "*_", "")
		err := filepath.Walk(s.teststoreDir, func(currentPath string, info os.FileInfo, pathErr error) error {
			if pathErr != nil {
				return pathErr
			}
			if strings.Contains(info.Name(), substr) {
				path = currentPath
				return filepath.SkipDir
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("walk %q: %w", s.workdir, err)
		}
	}

	cmd := []string{
		"pdfinfo",
		filepath.Base(path),
	}

	output, err := execCommandInIntegrationToolsContainer(ctx, cmd, path)
	if err != nil {
		return fmt.Errorf("exec %q: %w", cmd, err)
	}

	output = strings.ReplaceAll(output, " ", "")
	re := regexp.MustCompile(`Pages:(\d+)`)
	matches := re.FindStringSubmatch(output)

	if len(matches) < 2 {
		return errors.New("expected pages")
	}

	actual, err := strconv.Atoi(matches[1])
	if err != nil {
		return fmt.Errorf("convert pages value %q to integer: %w", matches[1], err)
	}

	if actual != pages {
		return fmt.Errorf("expected %d pages, but actual is %d", pages, actual)
	}

	return nil
}

func (s *scenario) thePdfShouldHaveImages(ctx context.Context, name string, images int) error {
	path := fmt.Sprintf("%s/%s/%s", s.workdir, s.resp.Header().Get("Gotenberg-Trace"), name)

	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("PDF %q does not exist", path)
	}

	cmd := []string{
		"pdfimages",
		"-list",
		filepath.Base(path),
	}

	output, err := execCommandInIntegrationToolsContainer(ctx, cmd, path)
	if err != nil {
		return fmt.Errorf("exec %q: %w", cmd, err)
	}

	// pdfimages -list outputs a header (2 lines) then one line per image.
	lines := strings.Split(strings.TrimSpace(output), "\n")
	actual := 0
	if len(lines) > 2 {
		actual = len(lines) - 2
	}

	if actual != images {
		return fmt.Errorf("expected %d image(s), but actual is %d", images, actual)
	}

	return nil
}

func (s *scenario) thePdfShouldBeSetToLandscapeOrientation(ctx context.Context, name string, kind string) error {
	var path string
	if !strings.HasPrefix(name, "*_") {
		path = fmt.Sprintf("%s/%s/%s", s.workdir, s.resp.Header().Get("Gotenberg-Trace"), name)

		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			return fmt.Errorf("PDF %q does not exist", path)
		}
	} else {
		substr := strings.ReplaceAll(name, "*_", "")
		err := filepath.Walk(s.teststoreDir, func(currentPath string, info os.FileInfo, pathErr error) error {
			if pathErr != nil {
				return pathErr
			}
			if strings.Contains(info.Name(), substr) {
				path = currentPath
				return filepath.SkipDir
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("walk %q: %w", s.workdir, err)
		}
	}

	cmd := []string{
		"pdfinfo",
		filepath.Base(path),
	}

	output, err := execCommandInIntegrationToolsContainer(ctx, cmd, path)
	if err != nil {
		return fmt.Errorf("exec %q: %w", cmd, err)
	}

	output = strings.ReplaceAll(output, " ", "")
	re := regexp.MustCompile(`Pagesize:(\d+)x(\d+).*`)
	matches := re.FindStringSubmatch(output)

	if len(matches) < 3 {
		return errors.New("expected page size")
	}

	invert := kind == "should NOT"

	width, err := strconv.Atoi(matches[1])
	if err != nil {
		return fmt.Errorf("convert width value %q to integer: %w", matches[1], err)
	}

	height, err := strconv.Atoi(matches[2])
	if err != nil {
		return fmt.Errorf("convert height value %q to integer: %w", matches[2], err)
	}

	if invert && height < width {
		return fmt.Errorf("expected height %d to be greater than width %d", height, width)
	}

	if !invert && width < height {
		return fmt.Errorf("expected width %d to be greater than height %d", width, height)
	}

	return nil
}

func (s *scenario) thePdfShouldHaveTheFollowingContentAtPage(ctx context.Context, name, kind string, page int, expected *godog.DocString) error {
	var path string
	if !strings.HasPrefix(name, "*_") {
		path = fmt.Sprintf("%s/%s/%s", s.workdir, s.resp.Header().Get("Gotenberg-Trace"), name)

		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			return fmt.Errorf("PDF %q does not exist", path)
		}
	} else {
		substr := strings.ReplaceAll(name, "*_", "")
		err := filepath.Walk(s.teststoreDir, func(currentPath string, info os.FileInfo, pathErr error) error {
			if pathErr != nil {
				return pathErr
			}
			if strings.Contains(info.Name(), substr) {
				path = currentPath
				return filepath.SkipDir
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("walk %q: %w", s.workdir, err)
		}
	}

	cmd := []string{
		"pdftotext",
		"-f",
		fmt.Sprintf("%d", page),
		"-l",
		fmt.Sprintf("%d", page),
		filepath.Base(path),
		"-",
	}

	output, err := execCommandInIntegrationToolsContainer(ctx, cmd, path)
	if err != nil {
		return fmt.Errorf("exec %q: %w", cmd, err)
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

func (s *scenario) thePdfsShouldBeFlatten(ctx context.Context, kind, should string) error {
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

	invert := should == "should NOT"

	for _, path := range paths {
		cmd := []string{
			"verapdf",
			"-off",
			"--extract",
			"annotations",
			filepath.Base(path),
		}

		output, err := execCommandInIntegrationToolsContainer(ctx, cmd, path)
		if err != nil {
			return fmt.Errorf("exec %q: %w", cmd, err)
		}

		if invert && strings.Contains(output, "<featuresReport></featuresReport>") {
			return fmt.Errorf("PDF %q is flatten", path)
		}

		if !invert && !strings.Contains(output, "<featuresReport></featuresReport>") {
			return fmt.Errorf("PDF %q is not flatten", path)
		}
	}

	return nil
}

func (s *scenario) thePdfsShouldBeEncrypted(ctx context.Context, kind string, should string) error {
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
		return fmt.Errorf("walk %q: %w", dirPath, err)
	}

	invert := should == "should NOT"
	re := regexp.MustCompile(`CommandLineError:Incorrectpassword`)

	for _, path := range paths {
		cmd := []string{
			"pdfinfo",
			filepath.Base(path),
		}

		output, err := execCommandInIntegrationToolsContainer(ctx, cmd, path)
		if err != nil {
			return fmt.Errorf("exec %q: %w", cmd, err)
		}

		output = strings.ReplaceAll(output, " ", "")
		output = strings.ReplaceAll(output, "\n", "")
		matches := re.FindStringSubmatch(output)
		isEncrypted := len(matches) >= 1 && matches[0] == "CommandLineError:Incorrectpassword"

		if invert && isEncrypted {
			return fmt.Errorf("PDF %q is encrypted", path)
		}
		if !invert && !isEncrypted {
			return fmt.Errorf("PDF %q is not encrypted: %q", path, output)
		}
	}

	return nil
}

func (s *scenario) thePdfsShouldHaveEmbeddedFile(ctx context.Context, kind, should, embed string) error {
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
		return fmt.Errorf("walk %q: %w", dirPath, err)
	}

	invert := should == "should NOT"

	for _, path := range paths {
		cmd := []string{
			"verapdf",
			"--off",
			"--loglevel",
			"0",
			"--extract",
			"embeddedFile",
			filepath.Base(path),
		}

		output, err := execCommandInIntegrationToolsContainer(ctx, cmd, path)
		if err != nil {
			return fmt.Errorf("exec %q: %w", cmd, err)
		}

		found := strings.Contains(output, fmt.Sprintf("<fileName>%s</fileName>", embed))

		if invert && found {
			return fmt.Errorf("embed %q found", embed)
		}

		if !invert && !found {
			return fmt.Errorf("embed %q not found", embed)
		}
	}

	return nil
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	s := &scenario{}
	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		wd, err := os.Getwd()
		if err != nil {
			return ctx, fmt.Errorf("get current directory: %w", err)
		}
		s.workdir = fmt.Sprintf("%s/teststore/%s", wd, uuid.NewString())
		err = os.MkdirAll(s.workdir, 0o755)
		if err != nil {
			return ctx, fmt.Errorf("create working directory: %w", err)
		}
		return ctx, nil
	})
	ctx.Given(`^I have a default Gotenberg container$`, s.iHaveADefaultGotenbergContainer)
	ctx.Given(`^I have a Gotenberg container with the following environment variable\(s\):$`, s.iHaveAGotenbergContainerWithTheFollowingEnvironmentVariables)
	ctx.Given(`^I have a (webhook|static) server$`, s.iHaveAServer)
	ctx.When(`^I make a "(GET|HEAD)" request to Gotenberg at the "([^"]*)" endpoint$`, s.iMakeARequestToGotenberg)
	ctx.When(`^I make a "(GET|HEAD)" request to Gotenberg at the "([^"]*)" endpoint with the following header\(s\):$`, s.iMakeARequestToGotenbergWithTheFollowingHeaders)
	ctx.When(`^I make a "(POST)" request to Gotenberg at the "([^"]*)" endpoint with the following form data and header\(s\):$`, s.iMakeARequestToGotenbergWithTheFollowingFormDataAndHeaders)
	ctx.When(`^I make (\d+) concurrent "(POST)" requests to Gotenberg at the "([^"]*)" endpoint with the following form data and header\(s\):$`, s.iMakeConcurrentRequestsToGotenberg)
	ctx.When(`^I wait for the asynchronous request to the webhook$`, s.iWaitForTheAsynchronousRequestToWebhook)
	ctx.Then(`^the Gotenberg container (should|should NOT) log the following entries:$`, s.theGotenbergContainerShouldLogTheFollowingEntries)
	ctx.Then(`^the response status code should be (\d+)$`, s.theResponseStatusCodeShouldBe)
	ctx.Then(`^all concurrent response status codes should be (\d+)$`, s.allConcurrentResponseStatusCodesShouldBe)
	ctx.Then(`^all concurrent responses should have (\d+) PDF\(s\)$`, s.allConcurrentResponsesShouldHavePdfs)
	ctx.Then(`^the (response|webhook request|file request|server request) header "([^"]*)" should be "([^"]*)"$`, s.theHeaderValueShouldBe)
	ctx.Then(`^the (response|webhook request|file request|server request) cookie "([^"]*)" should be "([^"]*)"$`, s.theCookieValueShouldBe)
	ctx.Then(`^the (response|webhook request) body should match string:$`, s.theBodyShouldMatchString)
	ctx.Then(`^the (response|webhook request) body should contain string:$`, s.theBodyShouldContainString)
	ctx.Then(`^the (response|webhook request) body should match JSON:$`, s.theBodyShouldMatchJSON)
	ctx.Then(`^the webhook event should match JSON:$`, s.theWebhookEventShouldMatchJSON)
	ctx.Then(`^there should be (\d+) PDF\(s\) in the (response|webhook request)$`, s.thereShouldBePdfs)
	ctx.Then(`^there should be the following file\(s\) in the (response|webhook request):$`, s.thereShouldBeTheFollowingFiles)
	ctx.Then(`^the (response|webhook request) PDF\(s\) should be valid "([^"]*)" with a tolerance of (\d+) failed rule\(s\)$`, s.thePdfsShouldBeValidWithAToleranceOf)
	ctx.Then(`^the (response|webhook request) PDF\(s\) (should|should NOT) be flatten$`, s.thePdfsShouldBeFlatten)
	ctx.Then(`^the (response|webhook request) PDF\(s\) (should|should NOT) be encrypted`, s.thePdfsShouldBeEncrypted)
	ctx.Then(`^the (response|webhook request) PDF\(s\) (should|should NOT) have the "([^"]*)" file embedded$`, s.thePdfsShouldHaveEmbeddedFile)
	ctx.Then(`^the "([^"]*)" PDF should have (\d+) page\(s\)$`, s.thePdfShouldHavePages)
	ctx.Then(`^the "([^"]*)" PDF (should|should NOT) be set to landscape orientation$`, s.thePdfShouldBeSetToLandscapeOrientation)
	ctx.Then(`^the "([^"]*)" PDF (should|should NOT) have the following content at page (\d+):$`, s.thePdfShouldHaveTheFollowingContentAtPage)
	ctx.Then(`^the "([^"]*)" PDF should have (\d+) image\(s\)$`, s.thePdfShouldHaveImages)
	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		if s.gotenbergContainer != nil {
			errTerminate := s.gotenbergContainer.Terminate(ctx, testcontainers.StopTimeout(0))
			if errTerminate != nil {
				return ctx, fmt.Errorf("terminate Gotenberg container: %w", errTerminate)
			}
		}
		if s.gotenbergContainerNetwork != nil {
			errRemove := s.gotenbergContainerNetwork.Remove(ctx)
			if errRemove != nil {
				return ctx, fmt.Errorf("remove Gotenberg container network: %w", errRemove)
			}
		}
		return ctx, nil
	})
	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		if err != nil {
			recordFailedScenario(sc)
		}

		errReset := s.reset(ctx)
		if errReset != nil {
			return ctx, fmt.Errorf("reset scenario: %w", errReset)
		}
		return ctx, nil
	})
}
