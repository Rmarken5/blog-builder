package build

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/tdewolff/minify/v2/minify"
)

// Note there's a new line before and after. Backticks do not honor \n characters
const cssTagTemplate = `
<link rel="stylesheet" href="%s" />
`

var _ CSSHandler = HandleCSS{}

var ErrNoHeadTag = errors.New("no head tag in html")

type (
	CSSHandler interface {
		InjectCSSIntoHTML(ctx context.Context, html io.Reader, cssPath string) ([]byte, error)
		GetCSSFilesFromSource(ctx context.Context) ([]ReaderWithPath, error)
		MinifyCSS(ctx context.Context, r io.Reader) ([]byte, error)
		WriteCSS(ctx context.Context, w io.Writer, minifiedCSS []byte) error
		CreateBuildFileFromCSSSource(ctx context.Context, path string) (*os.File, error)
		GetCSSDirectoryStructure(ctx context.Context) ([]string, error)
		GetBuiltCSSFiles(ctx context.Context) ([]ReaderWithPath, error)
		CreateBuildDirectoryForPath(context.Context, string) (string, error)
	}

	HandleCSS struct {
		cssBuildDirectory  string
		cssSourceDirectory string
		extension          string
	}
)

func NewHandleCSS(cssSourceDirectory, cssBuildDirectory, extension string) *HandleCSS {
	return &HandleCSS{
		cssBuildDirectory:  cssBuildDirectory,
		cssSourceDirectory: cssSourceDirectory,
		extension:          extension,
	}
}

func (c HandleCSS) InjectCSSIntoHTML(ctx context.Context, html io.Reader, cssPath string) ([]byte, error) {
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

func (c HandleCSS) GetCSSFilesFromSource(ctx context.Context) ([]ReaderWithPath, error) {
	return getFilesFromDirectory(c.cssSourceDirectory, c.extension)
}

func (c HandleCSS) MinifyCSS(ctx context.Context, r io.Reader) ([]byte, error) {
	cssContents, err := io.ReadAll(r)
	if err != nil {
		slog.Error("not able to read css", "error", err)
		return nil, err
	}
	minifiedCSS, err := minify.CSS(string(cssContents))
	if err != nil {
		slog.Error("error minifying css", "error", err)
		return nil, err
	}

	return []byte(minifiedCSS), nil
}

func (c HandleCSS) WriteCSS(_ context.Context, w io.Writer, minifiedCSS []byte) error {
	_, err := w.Write(minifiedCSS)
	if err != nil {
		slog.Error("error writing minified css", "error", err)
		return err
	}

	return nil
}

func (c HandleCSS) CreateBuildFileFromCSSSource(ctx context.Context, path string) (*os.File, error) {
	filePath := c.buildDirFromPath(path)
	cssFile, err := os.Create(filePath)
	if err != nil {
		slog.Error("error creating css file", "error", err)
		return nil, err
	}

	return cssFile, nil
}

func (c HandleCSS) buildDirFromPath(path string) string {
	return strings.Replace(path, c.cssSourceDirectory, c.cssBuildDirectory, 1)
}

func (c HandleCSS) GetCSSDirectoryStructure(ctx context.Context) ([]string, error) {
	return getDirectoryStructure(c.cssSourceDirectory)
}

func (c HandleCSS) GetBuiltCSSFiles(ctx context.Context) ([]ReaderWithPath, error) {
	return getFilesFromDirectory(c.cssBuildDirectory, c.extension)
}

func (c HandleCSS) CreateBuildDirectoryForPath(ctx context.Context, filePath string) (string, error) {
	fullPath := strings.Replace(filePath, c.cssSourceDirectory, c.cssBuildDirectory, 1)
	err := os.Mkdir(fullPath, 0777)
	if os.IsExist(err) {
		slog.Info("Directory already exists", "dir", fullPath)
		return fullPath, nil
	}
	if err != nil {
		log.Printf("error creating directory build: %s", fullPath)
		return "", err
	}
	return fullPath, nil
}
