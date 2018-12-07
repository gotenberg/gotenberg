package pm2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnoconv(t *testing.T) {
	p := &Unoconv{}
	err1 := p.Launch()
	require.Nil(t, err1)
	err2 := p.Shutdown()
	assert.Nil(t, err2)
}
