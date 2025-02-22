//go:build integration

package integration

import (
	"os"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"

	"github.com/gotenberg/gotenberg/v8/test/integration/scenario"
)

var opts = godog.Options{
	Format: "pretty",
	Paths:  []string{"features"},
	Output: colors.Colored(os.Stdout),
}

func TestMain(m *testing.M) {
	code := godog.TestSuite{
		Name:                "integration",
		ScenarioInitializer: scenario.InitializeScenario,
		Options:             &opts,
	}.Run()

	os.Exit(code)
}
