package logic

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

const cssTagTemplate = `<link rel="stylesheet" href="%s" />`

var ErrNoHeadTag = errors.New("no head tag in html")

type (
	CSSHandler interface {
		InjectCSSIntoHTML(html io.Reader, cssPath string) ([]byte, error)
	}

	HandleCSS struct {
	}
)

func (h HandleCSS) InjectCSSIntoHTML(html io.Reader, cssPath string) ([]byte, error) {
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
