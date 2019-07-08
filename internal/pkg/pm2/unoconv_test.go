package pm2

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestUnoconvStart(t *testing.T) {
	p := NewUnoconv(test.CreateTestLogger())
	err := p.Start()
	require.Nil(t, err)
}

func TestUnoconvShutdown(t *testing.T) {
	p := NewUnoconv(test.CreateTestLogger())
	err := p.Shutdown()
	require.Nil(t, err)
}
