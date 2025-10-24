package logic

import (
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type (
	Build interface {
		BuildHTMLFromMD(path string) error
	}
	HTMLBuilder struct {
		markdownPath    string
		buildOutputPath string
	}
)

func New(buildOutputPath string) *HTMLBuilder {
	return &HTMLBuilder{
		buildOutputPath: buildOutputPath,
	}
}

func (h HTMLBuilder) BuildHTMLFromMD(path string) error {
	h.markdownPath = path
	err := filepath.WalkDir(path, h.walker)
	if err != nil {
		slog.Error("program failed in error", "error", err)
		return err
	}

	return nil
}

func (h HTMLBuilder) walker(path string, dirEntry fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	slog.Info("path being walked", "path", path)
	if dirEntry.IsDir() {
		_, err := h.createNewBuildDirectory(path)
		if err != nil {
			log.Printf("unable to create new build directory: %s - %s", path, err.Error())
			return err
		}
		return nil
	}

	// Ignore anything that's not markdown
	if path[len(path)-3:] != ".md" {
		slog.Info("Ignoring path", "path", path)
		return nil
	}

	mdBytes, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Unable to read file: %s - %s", path, err.Error())
		return err
	}

	htmlFile, err := createHtmlFile(path)
	if err != nil {
		return err
	}
	defer htmlFile.Close()

	toHTML := mdToHTML(mdBytes)
	_, err = htmlFile.Write(toHTML)
	if err != nil {
		slog.Error("error writing html to file", "error", err)
		return err
	}

	return nil
}

func createHtmlFile(path string) (*os.File, error) {
	filePath := buildDirFromPath(path)
	filePath = strings.Replace(filePath, ".md", ".html", -1)
	htmlFile, err := os.Create(filePath)
	if err != nil {
		slog.Error("error creating html file", "error", err)
		return nil, err
	}

	return htmlFile, nil
}

func buildDirFromPath(path string) string {
	return strings.Replace(path, "markdown", "build", 1)
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

func (h HTMLBuilder) createNewBuildDirectory(filePath string) (string, error) {
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
