//go:build integration

package integration

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	flag "github.com/spf13/pflag"

	"github.com/gotenberg/gotenberg/v8/test/integration/scenario"
)

func TestMain(m *testing.M) {
	repository := flag.String("gotenberg-docker-repository", "", "")
	version := flag.String("gotenberg-version", "", "")
	platform := flag.String("gotenberg-container-platform", "", "")
	noConcurrency := flag.Bool("no-concurrency", false, "")
	tags := flag.String("tags", "", "")
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
	scenario.GotenbergContainerPlatform = *platform

	concurrency := runtime.NumCPU()
	if *noConcurrency {
		concurrency = 0
	}

	maxAttempts := 4 // 1 initial run + up to 3 retries
	paths := []string{"features"}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Reset the failure collector before each run.
		scenario.ResetFailedScenarios()

		code := godog.TestSuite{
			Name:                "integration",
			ScenarioInitializer: scenario.InitializeScenario,
			Options: &godog.Options{
				Format:      "pretty",
				Paths:       paths,
				Output:      colors.Colored(os.Stdout),
				Concurrency: concurrency,
				Tags:        *tags,
			},
		}.Run()

		if code == 0 {
			os.Exit(0)
		}

		failedPaths := scenario.FailedScenarioPaths()
		if len(failedPaths) == 0 || attempt == maxAttempts {
			os.Exit(code)
		}

		fmt.Fprintf(colors.Colored(os.Stdout),
			"\n\n--- %d scenario(s) failed, retrying (%d retries left) ---\n\n",
			len(failedPaths), maxAttempts-attempt,
		)

		// Next run: only the failed scenarios via file:line paths.
		paths = failedPaths
	}

	os.Exit(1)
}
