package pm2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnoconvLaunch(t *testing.T) {
	p := &Unoconv{}
	err := p.Launch()
	require.Nil(t, err)
}

func TestUnoconvShutdown(t *testing.T) {
	p := &Unoconv{}
	err := p.Shutdown()
	require.Nil(t, err)
}
