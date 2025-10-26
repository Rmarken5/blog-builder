package logic

import (
	"io"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

type (
	HTMLHandler interface {
		BuildHTMLFromMD(path string) error
		GetHTMLFilesFromBuildPath(path string) ([]ReaderwWithPath, error)
		ConvertMDToHTML(r io.Reader) ([]byte, error)
		WriteHTML(w io.Writer, data []byte) error
		CreateFileFromMDPath(path string) (*os.File, error)
	}
	HandleHTML struct {
		fileExtension   string
		markdownPath    string
		buildOutputPath string
		cssHandler      CSSHandler
	}
)

func NewHandleHTML(buildOutputPath string, markdownDirectory string) *HandleHTML {
	return &HandleHTML{
		markdownPath:    markdownDirectory,
		fileExtension:   "html",
		buildOutputPath: buildOutputPath,
	}
}

func (h HandleHTML) WriteHTML(w io.Writer, data []byte) error {
	_, err := w.Write(data)
	if err != nil {
		slog.Error("error writing data", err)
		return err
	}
	return nil
}

func (h HandleHTML) ConvertMDToHTML(r io.Reader) ([]byte, error) {
	mdBytes, err := io.ReadAll(r)
	if err != nil {
		slog.Error("error reading md", "error", err)
	}

	htmlBytes := mdToHTML(mdBytes)
	return htmlBytes, nil
}

func (h HandleHTML) BuildHTMLFromMD(path string) error {
	//h.markdownPath = path
	//err := filepath.WalkDir(path, h.walker)
	//if err != nil {
	//	slog.Error("program failed in error", "error", err)
	//	return err
	//}
	//
	return nil
}

//
//func (h HandleHTML) walker(path string, dirEntry fs.DirEntry, err error) error {
//	if err != nil {
//		return err
//	}
//	slog.Info("path being walked", "path", path)
//	if dirEntry.IsDir() {
//		_, err := h.createNewBuildDirectory(path)
//		if err != nil {
//			log.Printf("unable to create new build directory: %s - %s", path, err.Error())
//			return err
//		}
//		return nil
//	}
//
//	// Ignore anything that's not markdown
//	if path[len(path)-3:] != ".md" {
//		slog.Info("Ignoring path", "path", path)
//		return nil
//	}
//
//	mdBytes, err := os.ReadFile(path)
//	if err != nil {
//		log.Printf("Unable to read file: %s - %s", path, err.Error())
//		return err
//	}
//
//	htmlFile, err := createHtmlFile(path)
//	if err != nil {
//		return err
//	}
//	defer htmlFile.Close()
//
//	toHTML := mdToHTML(mdBytes)
//	_, err = htmlFile.Write(toHTML)
//	if err != nil {
//		slog.Error("error writing html to file", "error", err)
//		return err
//	}
//
//	return nil
//}

func (h HandleHTML) CreateFileFromMDPath(path string) (*os.File, error) {
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

func (h HandleHTML) createNewBuildDirectory(filePath string) (string, error) {
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

func (h HandleHTML) GetHTMLFilesFromBuildPath(rootPath string) ([]ReaderwWithPath, error) {
	return getFilesFromDirectory(rootPath, h.fileExtension)
}
