//go:build integration

package integration

import (
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

	code := godog.TestSuite{
		Name:                "integration",
		ScenarioInitializer: scenario.InitializeScenario,
		Options: &godog.Options{
			Format:      "pretty",
			Paths:       []string{"features"},
			Output:      colors.Colored(os.Stdout),
			Concurrency: concurrency,
		},
	}.Run()

	os.Exit(code)
}
