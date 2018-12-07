package pm2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnoconv(t *testing.T) {
	p := &Unoconv{}
	err := p.Launch()
	require.Nil(t, err)
	err = p.Shutdown(false)
	assert.Nil(t, err)
}
