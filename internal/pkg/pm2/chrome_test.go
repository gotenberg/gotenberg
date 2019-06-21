package pm2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChromeStart(t *testing.T) {
	p := NewChrome(false)
	err := p.Start()
	require.Nil(t, err)
}

func TestChromeShutdown(t *testing.T) {
	p := NewChrome(false)
	err := p.Shutdown()
	require.Nil(t, err)
}
