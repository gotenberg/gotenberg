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

	"github.com/cucumber/godog"
	"github.com/google/uuid"
	"github.com/mholt/archives"
	"github.com/testcontainers/testcontainers-go"
)

type gotenbergContainerCtxKey struct{}

type scenario struct {
	resp    *httptest.ResponseRecorder
	workdir string
}

func (s *scenario) reset() error {
	s.resp = httptest.NewRecorder()

	if s.workdir == "" {
		return nil
	}

	err := os.RemoveAll(s.workdir)
	if err != nil {
		return fmt.Errorf("remove workdir %q: %w", s.workdir, err)
	}
	s.workdir = ""

	return nil
}

func (s *scenario) iHaveADefaultGotenbergContainer(ctx context.Context) (context.Context, error) {
	c, err := startGotenbergContainer(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("create default Gotenberg container: %s", err)
	}
	return context.WithValue(ctx, gotenbergContainerCtxKey{}, c), nil
}

func (s *scenario) iHaveAGotenbergContainerWithTheFollowingEnvironmentVariables(ctx context.Context, envTable *godog.Table) (context.Context, error) {
	env := make(map[string]string)
	for _, row := range envTable.Rows {
		env[row.Cells[0].Value] = row.Cells[1].Value
	}
	c, err := startGotenbergContainer(ctx, env)
	if err != nil {
		return nil, fmt.Errorf("create Gotenberg container: %s", err)
	}
	return context.WithValue(ctx, gotenbergContainerCtxKey{}, c), nil
}

func (s *scenario) iMakeARequestToGotenberg(ctx context.Context, method, endpoint string) error {
	base, err := containerHttpEndpoint(ctx, ctx.Value(gotenbergContainerCtxKey{}).(testcontainers.Container), "3000")
	if err != nil {
		return fmt.Errorf("get container HTTP endpoint: %w", err)
	}

	resp, err := doRequest(method, fmt.Sprintf("%s%s", base, endpoint), nil)
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

			files[name] = append(files[name], fmt.Sprintf("%s/%s", wd, value))
		case "header":
			headers[name] = value
		default:
			return fmt.Errorf("unexpected %q %q", kind, value)
		}
	}

	base, err := containerHttpEndpoint(ctx, ctx.Value(gotenbergContainerCtxKey{}).(testcontainers.Container), "3000")
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

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get current directory: %w", err)
	}

	if s.workdir == "" {
		s.workdir = fmt.Sprintf("%s/teststore/%s", wd, uuid.NewString())
	}

	err = os.MkdirAll(fmt.Sprintf("%s/%s", s.workdir, resp.Header.Get("Gotenberg-Trace")), 0o755)
	if err != nil {
		return fmt.Errorf("create working directory: %w", err)
	}

	fpath := fmt.Sprintf("%s/%s/%s", s.workdir, resp.Header.Get("Gotenberg-Trace"), filename)
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

			targetPath := fmt.Sprintf("%s/%s/%s", s.workdir, resp.Header.Get("Gotenberg-Trace"), f.Name())
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

func (s *scenario) theGotenbergContainerShouldLogTheFollowingEntries(ctx context.Context, should string, entriesTable *godog.Table) error {
	c := ctx.Value(gotenbergContainerCtxKey{}).(testcontainers.Container)
	expected := make([]string, len(entriesTable.Rows))
	for i, row := range entriesTable.Rows {
		expected[i] = row.Cells[0].Value
	}

	logs, err := containerLogEntries(ctx, c)
	if err != nil {
		return fmt.Errorf("get log entries: %w", err)
	}

	invert := should == "should NOT"
	for _, entry := range expected {
		if !invert && !strings.Contains(logs, entry) {
			return fmt.Errorf("expected log entry %q not found", expected)
		}

		if invert && strings.Contains(logs, entry) {
			return fmt.Errorf("log entry %q NOT expected", expected)
		}
	}

	return nil
}

func (s *scenario) theResponseStatusCodeShouldBe(expected int) error {
	if expected != s.resp.Code {
		return fmt.Errorf("expected response status code to be: %d, but actual is: %d", expected, s.resp.Code)
	}
	return nil
}

func (s *scenario) theResponseHeaderValueShouldBe(name, expected string) error {
	actual := s.resp.Header().Get(name)
	if expected != actual {
		return fmt.Errorf("expected response header %q to be: %q, but actual is: %q", name, expected, actual)
	}
	return nil
}

func (s *scenario) theResponseBodyShouldMatchString(expected *godog.DocString) error {
	actual := s.resp.Body.String()
	if actual != expected.Content {
		return fmt.Errorf("expected response body to be: %q, but actual is: %q", expected.Content, actual)
	}
	return nil
}

func (s *scenario) theResponseBodyShouldMatchJSON(expectedDoc *godog.DocString) error {
	var expected, actual interface{}

	err := json.Unmarshal([]byte(expectedDoc.Content), &expected)
	if err != nil {
		return fmt.Errorf("unmarshal expected JSON: %w", err)
	}

	err = json.Unmarshal(s.resp.Body.Bytes(), &actual)
	if err != nil {
		return fmt.Errorf("unmarshal actual JSON: %w", err)
	}

	err = compareJson(expected, actual)
	if err != nil {
		return fmt.Errorf("expected matching JSON: %w", err)
	}

	return nil
}

func (s *scenario) thereShouldBePdfs(expected int) error {
	if s.workdir == "" {
		return errors.New("no resulting files")
	}

	var paths []string
	err := filepath.Walk(fmt.Sprintf("%s/%s", s.workdir, s.resp.Header().Get("Gotenberg-Trace")), func(path string, info os.FileInfo, pathErr error) error {
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

func (s *scenario) thereShouldBeTheFollowingFiles(filesTable *godog.Table) error {
	if s.workdir == "" {
		return errors.New("no resulting files")
	}

	var filenames []string
	err := filepath.Walk(fmt.Sprintf("%s/%s", s.workdir, s.resp.Header().Get("Gotenberg-Trace")), func(path string, info os.FileInfo, pathErr error) error {
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
		for _, filename := range filenames {
			if strings.EqualFold(row.Cells[0].Value, filename) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("expected file %q not found among %q", row.Cells[0].Value, filenames)
		}
	}

	return nil
}

func (s *scenario) theResponsePdfsShouldBeValidWithAToleranceOf(ctx context.Context, validate string, tolerance int) error {
	if s.workdir == "" {
		return errors.New("no resulting files")
	}

	var paths []string
	err := filepath.Walk(fmt.Sprintf("%s/%s", s.workdir, s.resp.Header().Get("Gotenberg-Trace")), func(path string, info os.FileInfo, pathErr error) error {
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

		re := regexp.MustCompile(`failedRules="(\d+)"`)
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

func InitializeScenario(ctx *godog.ScenarioContext) {
	s := &scenario{}
	ctx.Given(`^I have a default Gotenberg container$`, s.iHaveADefaultGotenbergContainer)
	ctx.Given(`^I have a Gotenberg container with the following environment variables:$`, s.iHaveAGotenbergContainerWithTheFollowingEnvironmentVariables)
	ctx.When(`^I make a "(GET|HEAD)" request to Gotenberg at the "([^"]*)" endpoint$`, s.iMakeARequestToGotenberg)
	ctx.When(`^I make a "(POST)" request to Gotenberg at the "([^"]*)" endpoint with the following form data and headers:$`, s.iMakeARequestToGotenbergWithTheFollowingFormDataAndHeaders)
	ctx.Then(`^the Gotenberg container "(should|should NOT)" log the following entries:$`, s.theGotenbergContainerShouldLogTheFollowingEntries)
	ctx.Then(`^the response status code should be (\d+)$`, s.theResponseStatusCodeShouldBe)
	ctx.Then(`^the response header "([^"]*)" should be "([^"]*)"$`, s.theResponseHeaderValueShouldBe)
	ctx.Then(`^the response body should match string:$`, s.theResponseBodyShouldMatchString)
	ctx.Then(`^the response body should match JSON:$`, s.theResponseBodyShouldMatchJSON)
	ctx.Then(`^there should be (\d+) PDF\(s\)$`, s.thereShouldBePdfs)
	ctx.Then(`^there should be the following file\(s\):$`, s.thereShouldBeTheFollowingFiles)
	ctx.Then(`^the response PDF\(s\) should be valid "([^"]*)" with a tolerance of (\d+) failed rule\(s\)$`, s.theResponsePdfsShouldBeValidWithAToleranceOf)
	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		c, ok := ctx.Value(gotenbergContainerCtxKey{}).(testcontainers.Container)
		if !ok {
			return ctx, nil
		}
		errTerminate := c.Terminate(ctx, testcontainers.StopTimeout(0))
		if errTerminate != nil {
			return ctx, fmt.Errorf("terminate Gotenberg container: %w", errTerminate)
		}
		return ctx, nil
	})
	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		errReset := s.reset()
		if errReset != nil {
			return ctx, fmt.Errorf("reset scenario: %w", errReset)
		}
		return ctx, nil
	})
}
