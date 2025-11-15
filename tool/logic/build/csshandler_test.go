package build

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/bradleyjkemp/cupaloy/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testHTMLNoLines = `<html><head><head/><body></body></html>`
const testHTMLNoLinesWant = `<html><head><link rel="stylesheet" href="./hello" /><head/><body></body></html>`
const testHTMLLines = `<html>
<head><head/>
<body></body>
</html>`

const testHTMLLinesWant = `<html>
<head>
<link rel="stylesheet" href="./hello" />
<head/>
<body></body>
</html>`

func TestHandleCSS_InjectCSSIntoHTML(t *testing.T) {
	t.Run("should inject css tag lines", func(t *testing.T) {

		reader := strings.NewReader(testHTMLLines)
		htmlBytes, err := HandleCSS{}.InjectCSSIntoHTML(context.Background(), reader, "./hello")
		assert.NoError(t, err)
		assert.Equal(t, testHTMLLinesWant, string(htmlBytes))
	})
	t.Run("should inject css tag no lines", func(t *testing.T) {

		reader := strings.NewReader(testHTMLNoLines)
		htmlBytes, err := HandleCSS{}.InjectCSSIntoHTML(context.Background(), reader, "./hello")
		assert.NoError(t, err)
		assert.Equal(t, testHTMLNoLinesWant, string(htmlBytes))
	})
}

func TestHandleCSS_MinifyCSS(t *testing.T) {
	t.Run("should minify css", func(t *testing.T) {
		file, err := os.ReadFile("./test-files/test.css")
		require.NoError(t, err)
		r := bytes.NewReader(file)

		css, err := HandleCSS{}.MinifyCSS(context.Background(), r)
		require.NoError(t, err)

		t.Log(string(css))

		cupaloy.SnapshotT(t, string(css))
	})
}
