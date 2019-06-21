package pm2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnoconvStart(t *testing.T) {
	p := NewUnoconv(false)
	err := p.Start()
	require.Nil(t, err)
}

func TestUnoconvShutdown(t *testing.T) {
	p := NewUnoconv(false)
	err := p.Shutdown()
	require.Nil(t, err)
}
