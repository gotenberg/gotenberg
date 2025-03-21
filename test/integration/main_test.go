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

var opts = godog.Options{
	Format:      "pretty",
	Paths:       []string{"features"},
	Output:      colors.Colored(os.Stdout),
	Concurrency: runtime.NumCPU(),
}

func TestMain(m *testing.M) {
	repository := flag.String("gotenberg-docker-repository", "", "")
	version := flag.String("gotenberg-version", "", "")
	platform := flag.String("gotenberg-container-platform", "", "")
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

	code := godog.TestSuite{
		Name:                "integration",
		ScenarioInitializer: scenario.InitializeScenario,
		Options:             &opts,
	}.Run()

	os.Exit(code)
}
