package printer

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thecodingmachine/gotenberg/internal/pkg/pm2"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestOffice(t *testing.T) {
	p := &pm2.Unoconv{}
	err := p.Launch()
	require.Nil(t, err)
	defer p.Shutdown(false)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	office := &Office{
		Context: ctx,
		FilePaths: []string{
			test.OfficeTestFilePath(t, "document.docx"),
		},
	}
	err = office.Print("foo.pdf")
	require.Nil(t, err)
	require.FileExists(t, "foo.pdf")
	err = os.Remove("foo.pdf")
	assert.Nil(t, err)
}
