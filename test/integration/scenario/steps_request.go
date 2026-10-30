package scenario

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cucumber/godog"
)

func (s *scenario) iHaveADefaultGotenbergContainer(ctx context.Context) error {
	c, err := startGotenbergContainer(ctx, nil)
	if err != nil {
		return fmt.Errorf("create Gotenberg container: %s", err)
	}
	s.gotenbergContainer = c
	return nil
}

func (s *scenario) iHaveAGotenbergContainerWithTheFollowingEnvironmentVariables(ctx context.Context, envTable *godog.Table) error {
	env := make(map[string]string)
	for _, row := range envTable.Rows {
		value, err := s.expand(row.Cells[1].Value)
		if err != nil {
			return err
		}
		env[row.Cells[0].Value] = value
	}
	c, err := startGotenbergContainer(ctx, env)
	if err != nil {
		return fmt.Errorf("create Gotenberg container: %s", err)
	}
	s.gotenbergContainer = c
	return nil
}

func (s *scenario) iHaveAServer(ctx context.Context) error {
	// One server serves every request of a scenario. Starting another would
	// leak the first one's listener and goroutines.
	if s.server != nil {
		return nil
	}

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

// expand replaces the helper server tokens of a table cell: {helper} with the
// URL Gotenberg reaches the helper server at, and {helper-host} with its host
// and port. A cell with a token needs the helper server started first, which
// for an environment variable means before the Gotenberg container.
func (s *scenario) expand(value string) (string, error) {
	if !strings.Contains(value, "{helper") {
		return value, nil
	}

	if s.server == nil {
		return "", fmt.Errorf("%q needs the helper server: add \"I have a webhook server\" before this step", value)
	}

	host := fmt.Sprintf("host.docker.internal:%d", s.hostPort)
	value = strings.ReplaceAll(value, "{helper-host}", host)
	value = strings.ReplaceAll(value, "{helper}", "http://"+host)

	return value, nil
}

// parseRequestTable reads a form data and header(s) table: rows of name,
// value and kind, where kind is "field", "file" or "header". Values may carry
// helper server tokens, see [scenario.expand]. A file path is relative to the
// integration directory, or to the output directory of the previous request
// when it starts with "teststore".
func (s *scenario) parseRequestTable(dataTable *godog.Table) (fields, files map[string][]string, headers map[string]string, err error) {
	fields = make(map[string][]string)
	files = make(map[string][]string)
	headers = make(map[string]string)

	wd, err := os.Getwd()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get current directory: %w", err)
	}

	for _, row := range dataTable.Rows {
		name, kind := row.Cells[0].Value, row.Cells[2].Value

		value, err := s.expand(row.Cells[1].Value)
		if err != nil {
			return nil, nil, nil, err
		}

		switch kind {
		case "field":
			fields[name] = append(fields[name], value)
		case "file":
			if strings.Contains(value, "teststore") {
				if s.teststoreDir == "" {
					return nil, nil, nil, errors.New("no teststore directory available from previous requests")
				}
				_, err := os.Stat(s.teststoreDir)
				if os.IsNotExist(err) {
					return nil, nil, nil, fmt.Errorf("directory %q does not exist", s.teststoreDir)
				}
				value = strings.ReplaceAll(value, "teststore", s.teststoreDir)
			} else {
				value = fmt.Sprintf("%s/%s", wd, value)
			}
			files[name] = append(files[name], value)
		case "header":
			headers[name] = value
		default:
			return nil, nil, nil, fmt.Errorf("unexpected %q %q", kind, value)
		}
	}

	return fields, files, headers, nil
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
			value, err := s.expand(row.Cells[1].Value)
			if err != nil {
				return err
			}
			headers[row.Cells[0].Value] = value
		}
	}

	resp, err := doRequest(method, fmt.Sprintf("%s%s", base, endpoint), headers, nil)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}

	s.resp, _, err = recordResponse(resp)
	return err
}

func (s *scenario) iMakeARequestToGotenbergWithTheFollowingFormDataAndHeaders(ctx context.Context, method, endpoint string, dataTable *godog.Table) error {
	if s.gotenbergContainer == nil {
		return errors.New("no Gotenberg container")
	}

	fields, files, headers, err := s.parseRequestTable(dataTable)
	if err != nil {
		return err
	}

	base, err := containerHttpEndpoint(ctx, s.gotenbergContainer, "3000")
	if err != nil {
		return fmt.Errorf("get container HTTP endpoint: %w", err)
	}

	resp, err := doFormDataRequest(method, fmt.Sprintf("%s%s", base, endpoint), fields, files, headers)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}

	rec, body, err := recordResponse(resp)
	if err != nil {
		return err
	}
	s.resp = rec

	dirPath := fmt.Sprintf("%s/%s", s.workdir, rec.Header().Get("Gotenberg-Trace"))

	if rec.Code == http.StatusNoContent {
		// Gotenberg processes this asynchronously. The webhook test server
		// will save the incoming files under this trace ID directory shortly.
		s.teststoreDir = dirPath
		return nil
	}

	if rec.Code != http.StatusOK {
		return nil
	}

	filename, ok, err := outputFilename(rec.Header().Get("Content-Disposition"))
	if err != nil {
		return err
	}
	if !ok {
		if rec.Header().Get("Content-Disposition") != "" {
			return errors.New("no filename in Content-Disposition header")
		}
		return nil
	}

	s.teststoreDir = dirPath

	return storeOutput(ctx, dirPath, filename, rec.Header().Get("Content-Type"), body)
}

func (s *scenario) iMakeConcurrentRequestsToGotenberg(ctx context.Context, count int, method, endpoint string, dataTable *godog.Table) error {
	if s.gotenbergContainer == nil {
		return errors.New("no Gotenberg container")
	}

	fields, files, headers, err := s.parseRequestTable(dataTable)
	if err != nil {
		return err
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
			if reqErr == nil {
				var (
					rec  *httptest.ResponseRecorder
					body []byte
				)
				rec, body, reqErr = recordResponse(resp)
				if reqErr == nil {
					reqErr = storeConcurrentOutput(ctx, s.workdir, rec, body)
				}
				if reqErr == nil {
					mu.Lock()
					s.concurrentResps = append(s.concurrentResps, rec)
					mu.Unlock()
					return
				}
			}

			mu.Lock()
			errs = append(errs, reqErr)
			mu.Unlock()
		})
	}

	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("concurrent requests failed: %v", errs)
	}

	return nil
}

// storeConcurrentOutput stores the output of a successful concurrent request
// under its trace ID, so that each response is checked on its own.
func storeConcurrentOutput(ctx context.Context, workdir string, rec *httptest.ResponseRecorder, body []byte) error {
	if rec.Code != http.StatusOK {
		return nil
	}

	filename, ok, err := outputFilename(rec.Header().Get("Content-Disposition"))
	if err != nil || !ok {
		return err
	}

	dirPath := fmt.Sprintf("%s/%s", workdir, rec.Header().Get("Gotenberg-Trace"))
	return storeOutput(ctx, dirPath, filename, rec.Header().Get("Content-Type"), body)
}

// iMakeSequentialRequestsToGotenbergProbing mirrors the client loop from
// https://github.com/gotenberg/gotenberg/issues/1648: a conversion, then a
// probe, repeated. It records both the conversion and the probe responses so a
// scenario can assert that a planned process restart neither fails the probe
// nor breaks the conversions. Requests are sequential on purpose, since the bug
// only surfaces between two conversions.
func (s *scenario) iMakeSequentialRequestsToGotenbergProbing(ctx context.Context, count int, method, endpoint, probeEndpoint string, dataTable *godog.Table) error {
	if s.gotenbergContainer == nil {
		return errors.New("no Gotenberg container")
	}

	fields, files, headers, err := s.parseRequestTable(dataTable)
	if err != nil {
		return err
	}

	base, err := containerHttpEndpoint(ctx, s.gotenbergContainer, "3000")
	if err != nil {
		return fmt.Errorf("get container HTTP endpoint: %w", err)
	}

	s.probeResps = make([]*httptest.ResponseRecorder, 0, count)
	s.sequentialResps = make([]*httptest.ResponseRecorder, 0, count)

	for i := range count {
		resp, reqErr := doFormDataRequest(method, fmt.Sprintf("%s%s", base, endpoint), fields, files, headers)
		if reqErr != nil {
			return fmt.Errorf("request %d: do request: %w", i+1, reqErr)
		}

		rec, _, recErr := recordResponse(resp)
		if recErr != nil {
			return fmt.Errorf("request %d: %w", i+1, recErr)
		}
		s.resp = rec
		s.sequentialResps = append(s.sequentialResps, rec)

		probeResp, probeErr := doRequest(http.MethodGet, fmt.Sprintf("%s%s", base, probeEndpoint), nil, nil)
		if probeErr != nil {
			return fmt.Errorf("probe %d: do request: %w", i+1, probeErr)
		}

		probeRec, _, probeRecErr := recordResponse(probeResp)
		if probeRecErr != nil {
			return fmt.Errorf("probe %d: %w", i+1, probeRecErr)
		}
		s.probeResps = append(s.probeResps, probeRec)
	}

	return nil
}

// webhookWaitTimeout bounds the wait for an asynchronous webhook: the
// conversion, within --api-timeout (30s by default), then the delivery.
const webhookWaitTimeout = 90 * time.Second

func (s *scenario) iWaitForTheAsynchronousRequestToWebhook(ctx context.Context) error {
	if s.server == nil {
		return errors.New("server not initialized")
	}

	return s.server.waitForWebhook(ctx, webhookWaitTimeout)
}

// iStopTheGotenbergContainer sends SIGTERM to Gotenberg and expects a graceful
// shutdown: a clean exit within the stop timeout.
func (s *scenario) iStopTheGotenbergContainer(ctx context.Context) error {
	if s.gotenbergContainer == nil {
		return errors.New("no Gotenberg container")
	}

	timeout := 60 * time.Second
	err := s.gotenbergContainer.Stop(ctx, &timeout)
	if err != nil {
		return fmt.Errorf("stop Gotenberg container: %w", err)
	}

	state, err := s.gotenbergContainer.State(ctx)
	if err != nil {
		return fmt.Errorf("get Gotenberg container state: %w", err)
	}
	if state.ExitCode != 0 {
		return fmt.Errorf("expected Gotenberg to exit with code 0, got %d", state.ExitCode)
	}

	return nil
}
