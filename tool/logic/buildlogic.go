package logic

import (
	"bytes"
	"context"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type (
	PayloadBuilder interface {
		BuildPayload(ctx context.Context, inputPath, payloadPath string) error
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

func (b BuildPayload) BuildPayload(ctx context.Context, inputPath, payloadPath string) error {
	log.Println("creating build directory")

	buildPaths, err := b.markdownHandler.GetMarkdownDirectoryStructure(ctx, inputPath)
	if err != nil {
		slog.Error("error getting build paths from markdown directory", "error", err)
		return err
	}

	for _, p := range buildPaths {
		_, err := b.htmlHandler.CreateBuildDirectoryForPath(ctx, p)
		if err != nil {
			if !os.IsExist(err) {
				slog.Error("error making directory for build path", "error", err)
				return err
			}
		}
	}

	cssBuildPaths, err := b.cssHandler.GetCSSDirectoryStructure(ctx)
	if err != nil {
		slog.Error("error getting build paths from css directory", "error", err)
		return err
	}

	for _, p := range cssBuildPaths {
		_, err := b.cssHandler.CreateBuildDirectoryForPath(ctx, p)
		if err != nil {
			if !os.IsExist(err) {
				slog.Error("error making directory for css build path", "error", err)
				return err
			}
		}
	}

	cssFiles, err := b.cssHandler.GetCSSFilesFromPath(ctx)
	if err != nil {
		slog.Error("error getting css files from css directory", "error", err)
		return err
	}

	for _, cssFile := range cssFiles {
		minifiedBytes, err := b.cssHandler.MinifyCSS(ctx, cssFile.Reader)
		if err != nil {
			slog.Error("error minifying css file", "error", err)
			return err
		}

		outputFile, err := b.cssHandler.CreateBuildFileFromCSSSource(ctx, cssFile.Path)
		if err != nil {
			slog.Error("error creating css file", "error", err)
			return err
		}
		_, err = outputFile.Write(minifiedBytes)
		if err != nil {
			slog.Error("error writing minified bytes to build output", "error", err)
			return err
		}
		outputFile.Close()
	}

	cssBuildFiles, err := b.cssHandler.GetBuiltCSSFiles(ctx)
	if err != nil {
		slog.Error("error getting built css files", "error", err)
		return err
	}

	log.Printf("reading markdown from %s and building to %s", inputPath, payloadPath)
	markdownFiles, err := b.markdownHandler.GetMarkdownFilesFromPath(ctx, inputPath)
	if err != nil {
		slog.Error("error reading markdown directory", "error", err)
		return err
	}
	for _, mdFile := range markdownFiles {
		htmlBytes, err := b.htmlHandler.ConvertMDToHTML(ctx, mdFile.Reader)
		mdFile.Reader.Close()
		if err != nil {
			return err
		}

		for _, css := range cssBuildFiles {
			cssPath := strings.TrimPrefix(css.Path, "build/")
			count := strings.Count(mdFile.Path, "/")
			cssPath = strings.Repeat("../", count-1) + cssPath
			htmlBytes, err = b.cssHandler.InjectCSSIntoHTML(ctx, bytes.NewReader(htmlBytes), cssPath)
			if err != nil {
				slog.Error("error injecting css into html", "error", err)
				return err
			}
		}

		htmlFile, err := b.htmlHandler.CreateFileFromMDPath(ctx, mdFile.Path)
		if err != nil {
			slog.Error("error creating html file from markdown", "error", err)
			return err
		}

		err = b.htmlHandler.WriteHTML(ctx, htmlFile, htmlBytes)
		if err != nil {
			slog.Error("error writing to html to file")
		}

		htmlFile.Close()
	}

	for _, css := range cssBuildFiles {
		css.Reader.Close()
	}

	return nil
}

func getFilesFromDirectory(rootPath, extension string) ([]ReaderWithPath, error) {
	files := make([]ReaderWithPath, 0)
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

		files = append(files, ReaderWithPath{
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
