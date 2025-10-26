package logic

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// Note there's a new line before and after. Backticks do not honor \n characters
const cssTagTemplate = `
<link rel="stylesheet" href="%s" />
`

var ErrNoHeadTag = errors.New("no head tag in html")

type (
	CSSHandler interface {
		InjectCSSIntoHTML(html io.Reader, cssPath string) ([]byte, error)
		GetCSSFilesFromPath(path string) ([]ReaderwWithPath, error)
		MinifyCSS(r io.Reader) ([]byte, error)
		WriteCSS(w io.Writer, minifiedCSS []byte) error
	}

	HandleCSS struct {
		extension string
	}
)

func NewHandleCSS(extension string) *HandleCSS {
	return &HandleCSS{
		extension: extension,
	}
}

func (c HandleCSS) InjectCSSIntoHTML(html io.Reader, cssPath string) ([]byte, error) {
	htmlBytes, err := io.ReadAll(html)
	if err != nil {
		return []byte{}, err
	}

	htmlStr := string(htmlBytes)

	// Find the closing > of the <head> tag
	headStartIdx := strings.Index(htmlStr, "<head")
	if headStartIdx == -1 {
		return []byte{}, ErrNoHeadTag
	}

	// Find the > that closes the <head...> tag
	headEndIdx := strings.Index(htmlStr[headStartIdx:], ">")
	if headEndIdx == -1 {
		return []byte{}, ErrNoHeadTag
	}

	insertPos := headStartIdx + headEndIdx + 1

	// Insert CSS link after <head>
	result := htmlStr[:insertPos] +
		fmt.Sprintf(cssTagTemplate, cssPath) +
		htmlStr[insertPos:]

	return []byte(result), nil
}

func (c HandleCSS) GetCSSFilesFromPath(path string) ([]ReaderwWithPath, error) {
	return getFilesFromDirectory(path, c.extension)
}
