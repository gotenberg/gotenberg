//go:build corpus

package corpus

import (
	"testing"

	"github.com/gotenberg/gotenberg/v9/test/integration/scenario"
)

func TestMain(_ *testing.M) {
	scenario.RunSuite("corpus")
}
