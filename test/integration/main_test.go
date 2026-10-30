//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	flag "github.com/spf13/pflag"

	"github.com/gotenberg/gotenberg/v9/test/integration/scenario"
)

// runTimeout bounds a whole run, retries included. It stays below the go test
// timeout of the Makefile.
const runTimeout = 38 * time.Minute

func TestMain(m *testing.M) {
	repository := flag.String("gotenberg-docker-repository", "", "")
	version := flag.String("gotenberg-version", "", "")
	variant := flag.String("gotenberg-variant", "", "")
	platform := flag.String("gotenberg-container-platform", "", "")
	concurrencyFlag := flag.Int("concurrency", 0, "")
	tags := flag.String("tags", "", "")
	keepFailed := flag.String("keep-failed", "", "")
	flag.Parse()

	if *platform == "" {
		switch runtime.GOARCH {
		case "arm64":
			*platform = "linux/arm64"
		default:
			*platform = "linux/amd64"
		}
	}

	scenario.GotenbergDockerRepository = *repository
	scenario.GotenbergVersion = *version
	scenario.GotenbergVariant = *variant
	scenario.GotenbergContainerPlatform = *platform
	scenario.KeepFailedDir = *keepFailed

	concurrency := *concurrencyFlag
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}

	maxAttempts := 4 // 1 initial run + up to 3 retries
	paths := []string{"features"}

	// Steps run on this context: a step that waits on it fails before the go
	// test timeout (-timeout 40m in the Makefile) panics, which would skip the
	// retries and the report.
	timeoutCtx, cancelTimeout := context.WithTimeout(context.Background(), runTimeout)
	defer cancelTimeout()

	// An interrupt cancels the steps, then the run cleans up and stops rather
	// than retrying. A second interrupt kills the process.
	runCtx, stopSignals := signal.NotifyContext(timeoutCtx, os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	runDir, err := os.MkdirTemp("", "gotenberg-integration-")
	if err != nil {
		fmt.Fprintf(colors.Colored(os.Stdout), "create the run directory: %v\n", err)
		os.Exit(1)
	}
	scenario.RunDir = runDir

	// os.Exit skips the deferred calls: exit cleans up first.
	exit := func(code int) {
		stopSignals()
		err := os.RemoveAll(runDir)
		if err != nil {
			fmt.Fprintf(colors.Colored(os.Stdout), "warning: remove the run directory: %v\n", err)
		}
		os.Exit(code)
	}

	// Scenarios that failed at least once: reported when they pass on a retry.
	var flaky []string

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Reset the failure collector before each run.
		scenario.ResetFailedScenarios()

		// Reclaim subnets leaked by prior runs or failed container starts
		// before they exhaust Docker's predefined address pools.
		if deleted, err := scenario.PruneOrphanedNetworks(context.Background()); err != nil {
			fmt.Fprintf(colors.Colored(os.Stdout), "warning: prune orphaned networks: %v\n", err)
		} else if deleted > 0 {
			fmt.Fprintf(colors.Colored(os.Stdout), "pruned %d orphaned network(s)\n", deleted)
		}

		code := godog.TestSuite{
			Name:                "integration",
			ScenarioInitializer: scenario.InitializeScenario,
			Options: &godog.Options{
				Format:         "pretty",
				Paths:          paths,
				Output:         colors.Colored(os.Stdout),
				Concurrency:    concurrency,
				Tags:           *tags,
				DefaultContext: runCtx,
				// An undefined or pending step fails the run instead of
				// passing it unchecked.
				Strict: true,
			},
		}.Run()

		// Pooled containers live across scenarios, not across runs: a retry
		// starts from fresh ones.
		if err := scenario.TerminateDefaultContainers(context.Background()); err != nil {
			fmt.Fprintf(colors.Colored(os.Stdout), "warning: %v\n", err)
		}

		if runCtx.Err() != nil && timeoutCtx.Err() == nil {
			fmt.Fprintf(colors.Colored(os.Stdout), "\n--- interrupted ---\n")
			exit(130)
		}

		if code == 0 {
			if len(flaky) > 0 {
				fmt.Fprintf(colors.Colored(os.Stdout), "\n--- %d flaky scenario(s), passed on a retry ---\n%s\n", len(flaky), strings.Join(flaky, "\n"))
			}
			exit(0)
		}

		// A failure the retry cannot target would be dropped from it, and a
		// green retry would then hide it.
		if unmapped := scenario.UnmappedFailures(); unmapped > 0 {
			fmt.Fprintf(colors.Colored(os.Stdout), "\n--- %d failed scenario(s) cannot be retried, failing the run ---\n", unmapped)
			exit(code)
		}

		failedPaths := scenario.FailedScenarioPaths()
		if len(failedPaths) == 0 || attempt == maxAttempts {
			exit(code)
		}

		for _, path := range failedPaths {
			if !slices.Contains(flaky, path) {
				flaky = append(flaky, path)
			}
		}

		fmt.Fprintf(colors.Colored(os.Stdout),
			"\n\n--- %d scenario(s) failed, retrying (%d retries left) ---\n\n",
			len(failedPaths), maxAttempts-attempt,
		)

		// Next run: only the failed scenarios via file:line paths.
		paths = failedPaths
	}

	exit(1)
}
