package logic

import (
	"context"
	"io"
)

const markdownFileExtension = ".md"

var _ MarkdownHandler = HandleMarkdown{}

type (
	ReaderWithPath struct {
		Path   string
		Reader io.ReadCloser
	}

	MarkdownHandler interface {
		GetMarkdownFilesFromPath(ctx context.Context, path string) ([]ReaderWithPath, error)
		GetMarkdownDirectoryStructure(ctx context.Context, path string) ([]string, error)
	}

	HandleMarkdown struct {
	}
)

func NewHandleMarkdown() *HandleMarkdown {
	return &HandleMarkdown{}
}

func (m HandleMarkdown) GetMarkdownFilesFromPath(ctx context.Context, path string) ([]ReaderWithPath, error) {
	return getFilesFromDirectory(path, markdownFileExtension)
}

func (m HandleMarkdown) GetMarkdownDirectoryStructure(ctx context.Context, path string) ([]string, error) {
	return getDirectoryStructure(path)
}
