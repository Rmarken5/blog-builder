package logic

import (
	"context"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

var _ HTMLHandler = HandleHTML{}

const HTMLFileExtension = ".html"

type (
	HTMLHandler interface {
		GetHTMLFilesFromBuildPath(ctx context.Context, path string) ([]ReaderWithPath, error)
		ConvertMDToHTML(ctx context.Context, r io.Reader) ([]byte, error)
		WriteHTML(ctx context.Context, w io.Writer, data []byte) error
		CreateFileFromMDPath(ctx context.Context, path string) (*os.File, error)
		CreateBuildDirectoryForPath(ctx context.Context, path string) (string, error)
	}
	HandleHTML struct {
		fileExtension   string
		markdownPath    string
		buildOutputPath string
		cssHandler      CSSHandler
	}
)

func NewHandleHTML(markdownDirectory string, buildOutputPath string) *HandleHTML {
	return &HandleHTML{
		markdownPath:    markdownDirectory,
		fileExtension:   HTMLFileExtension,
		buildOutputPath: buildOutputPath,
	}
}

func (h HandleHTML) WriteHTML(ctx context.Context, w io.Writer, data []byte) error {
	_, err := w.Write(data)
	if err != nil {
		slog.Error("error writing data", "error", err)
		return err
	}
	return nil
}

func (h HandleHTML) ConvertMDToHTML(ctx context.Context, r io.Reader) ([]byte, error) {
	mdBytes, err := io.ReadAll(r)
	if err != nil {
		slog.Error("error reading md", "error", err)
	}

	htmlBytes := mdToHTML(mdBytes)
	return htmlBytes, nil
}

func (h HandleHTML) CreateFileFromMDPath(ctx context.Context, path string) (*os.File, error) {
	filePath := h.buildDirFromPath(path)
	filePath = strings.Replace(filePath, markdownFileExtension, h.fileExtension, -1)
	htmlFile, err := os.Create(filePath)
	if err != nil {
		slog.Error("error creating html file", "error", err)
		return nil, err
	}

	return htmlFile, nil
}

func (h HandleHTML) buildDirFromPath(path string) string {
	return strings.Replace(path, h.markdownPath, h.buildOutputPath, 1)
}

const generatorTag = `  <meta name="GENERATOR" content="github.com/rmarken5/blog-builder`

func mdToHTML(md []byte) []byte {
	// create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	// create HTML renderer with extensions
	htmlFlags := html.CommonFlags | html.HrefTargetBlank | html.CompletePage
	opts := html.RendererOptions{Flags: htmlFlags, Generator: generatorTag}
	renderer := html.NewRenderer(opts)

	return markdown.Render(doc, renderer)
}

func (h HandleHTML) CreateBuildDirectoryForPath(ctx context.Context, filePath string) (string, error) {
	fullPath := strings.Replace(filePath, h.markdownPath, h.buildOutputPath, 1)
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

func (h HandleHTML) GetHTMLFilesFromBuildPath(ctx context.Context, rootPath string) ([]ReaderWithPath, error) {
	return getFilesFromDirectory(rootPath, h.fileExtension)
}
