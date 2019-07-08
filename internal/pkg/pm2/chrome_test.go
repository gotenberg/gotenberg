package pm2

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestChromeStart(t *testing.T) {
	p := NewChrome(test.CreateTestLogger())
	err := p.Start()
	require.Nil(t, err)
}

func TestChromeShutdown(t *testing.T) {
	p := NewChrome(test.CreateTestLogger())
	err := p.Shutdown()
	require.Nil(t, err)
}
