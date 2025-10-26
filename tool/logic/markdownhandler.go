package logic

import "io"

const markdownFileExtension = ".md"

type (
	ReaderwWithPath struct {
		Path   string
		Reader io.ReadCloser
	}

	MarkdownHandler interface {
		GetMarkdownFilesFromPath(path string) ([]ReaderwWithPath, error)
		GetMarkdownDirectoryStructure(path string) ([]string, error)
	}

	HandleMarkdown struct {
	}
)

func NewHandleMarkdown() *HandleMarkdown {
	return &HandleMarkdown{}
}

func (m HandleMarkdown) GetMarkdownFilesFromPath(path string) ([]ReaderwWithPath, error) {
	return getFilesFromDirectory(path, markdownFileExtension)
}

func (m HandleMarkdown) GetMarkdownDirectoryStructure(path string) ([]string, error) {
	return getDirectoryStructure(path)
}
