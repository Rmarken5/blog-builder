package logic

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
		htmlBytes, err := HandleCSS{}.InjectCSSIntoHTML(reader, "./hello")
		assert.NoError(t, err)
		assert.Equal(t, testHTMLLinesWant, string(htmlBytes))
	})
	t.Run("should inject css tag no lines", func(t *testing.T) {

		reader := strings.NewReader(testHTMLNoLines)
		htmlBytes, err := HandleCSS{}.InjectCSSIntoHTML(reader, "./hello")
		assert.NoError(t, err)
		assert.Equal(t, testHTMLNoLinesWant, string(htmlBytes))
	})
}
