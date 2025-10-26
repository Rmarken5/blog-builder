package logic

import (
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type (
	PayloadBuilder interface {
		BuildPayload(inputPath, payloadPath string) error
	}
	BuildPayload struct {
		htmlHandler     HTMLHandler
		cssHandler      CSSHandler
		markdownHandler MarkdownHandler
	}
)

func NewPayloadBuilder(htmlHandler HTMLHandler, cssHandler CSSHandler, markdownHandler MarkdownHandler) *BuildPayload {
	return &BuildPayload{
		htmlHandler:     htmlHandler,
		cssHandler:      cssHandler,
		markdownHandler: markdownHandler,
	}
}

func (b BuildPayload) BuildPayload(inputPath, payloadPath string) error {
	log.Printf("reading markdown from %s and building to %s", inputPath, payloadPath)
	markdownFiles, err := b.markdownHandler.GetMarkdownFilesFromPath(inputPath)
	if err != nil {
		slog.Error("error reading markdown directory", "error", err)
		return err
	}
	for _, mdFile := range markdownFiles {
		htmlBytes, err := b.htmlHandler.ConvertMDToHTML(mdFile.Reader)
		mdFile.Reader.Close()
		if err != nil {
			return err
		}

		htmlFile, err := b.htmlHandler.CreateFileFromMDPath(mdFile.Path)
		if err != nil {
			slog.Error("error creating html file from markdown", "error", err)
			return err
		}

		err = b.htmlHandler.WriteHTML(htmlFile, htmlBytes)
		if err != nil {
			slog.Error("error writing to html to file")
		}
		htmlFile.Close()
	}

	return nil
}

func getFilesFromDirectory(rootPath, extension string) ([]ReaderwWithPath, error) {
	files := make([]ReaderwWithPath, 0)
	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {

		if d.IsDir() {
			return nil
		}

		if strings.ToLower(path[len(path)-len(extension):]) != strings.ToLower(extension) {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}

		files = append(files, ReaderwWithPath{
			Path:   path,
			Reader: f,
		})
		return nil
	})
	if err != nil {
		slog.Error("error getting files from path", "path", rootPath, "error", err)
		return nil, err
	}
	return files, nil
}

func getDirectoryStructure(rootPath string) ([]string, error) {
	dirs := make([]string, 0)
	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {

		if d.IsDir() {
			dirs = append(dirs, path)
		}

		return nil
	})
	if err != nil {
		slog.Error("error getting files from path", "path", rootPath, "error", err)
		return nil, err
	}
	return dirs, nil
}
