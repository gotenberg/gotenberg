package normalize

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	const (
		toNormalize string = "analise da aplicação"
		expected    string = "analise da aplicacao"
	)
	v, err := String(toNormalize)
	assert.Nil(t, err)
	assert.Equal(t, expected, v)
}
