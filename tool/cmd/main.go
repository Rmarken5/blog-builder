package main

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

func main() {
	markdownDir := "markdown"
	err := filepath.WalkDir(markdownDir, walker)
	if err != nil {
		slog.Error("program failed in error", "error", err)
	}
}

func walker(path string, dirEntry fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	slog.Info("path being walked", "path", path)
	if dirEntry.IsDir() {
		_, err := createNewBuildDirectory(path)
		if err != nil {
			log.Printf("unable to create new build directory: %s - %s", path, err.Error())
			return err
		}
		//dir, err := os.ReadDir(path)
		//if err != nil {
		//	slog.Error("reading directory: ", "path", path, "error", err)
		//	return err
		//}
		//for _, entry := range dir {
		//	err = filepath.WalkDir(path+"/"+entry.Name(), walker)
		//	if err != nil {
		//		log.Printf("Unable to walk path: %s - %s", path, err.Error())
		//		return err
		//	}
		//}

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

func mdToHTML(md []byte) []byte {
	// create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	// create HTML renderer with extensions
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return markdown.Render(doc, renderer)
}

func createNewBuildDirectory(filePath string) (string, error) {
	fullPath := strings.Replace(filePath, "markdown", "build", 1)
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
