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

func TestHTML(t *testing.T) {
	p := &pm2.Chrome{}
	err := p.Launch()
	require.Nil(t, err)
	defer p.Shutdown(false)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	html := &HTML{
		Context:      ctx,
		PaperWidth:   8.27,
		PaperHeight:  11.7,
		MarginTop:    1,
		MarginBottom: 1,
		MarginLeft:   1,
		MarginRight:  1,
	}
	html.WithLocalURL(test.HTMLTestFilePath(t, "index.html"))
	err = html.WithHeaderFile(test.HTMLTestFilePath(t, "header.html"))
	require.Nil(t, err)
	err = html.WithFooterFile(test.HTMLTestFilePath(t, "footer.html"))
	require.Nil(t, err)
	err = html.Print("foo.pdf")
	require.Nil(t, err)
	require.FileExists(t, "foo.pdf")
	err = os.Remove("foo.pdf")
	assert.Nil(t, err)
}
