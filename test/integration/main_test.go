//go:build integration

package integration

import (
	"testing"

	"github.com/gotenberg/gotenberg/v9/test/integration/scenario"
)

func TestMain(_ *testing.M) {
	scenario.RunSuite("integration")
}
