package pm2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChromeLaunch(t *testing.T) {
	p := &Chrome{}
	err := p.Launch()
	require.Nil(t, err)
}

func TestChromeShutdown(t *testing.T) {
	p := &Chrome{}
	err := p.Shutdown()
	require.Nil(t, err)
}
