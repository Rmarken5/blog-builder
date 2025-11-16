package build

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"strings"
	"time"
)

const (
	metadataBrace         = "---"
	tagBullet             = "  - "
	markdownFileExtension = ".md"
)

var _ MarkdownHandler = HandleMarkdown{}

type (
	TagFinder       func(s *bufio.Scanner) ([]string, error)
	CreatedAtFinder func(s *bufio.Scanner) (time.Time, error)
	ReaderWithPath  struct {
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

func GetTags(_ context.Context, r io.Reader, tagFinder TagFinder) ([]string, error) {
	scanner := bufio.NewScanner(r)
	tags, err := tagFinder(scanner)
	if err != nil {
		return nil, err
	}
	return tags, nil
}

func GetCreatedAtDate(ctx context.Context, r io.Reader, createdAtFinder CreatedAtFinder) (time.Time, error) {
	scanner := bufio.NewScanner(r)
	createdAt, err := createdAtFinder(scanner)
	if err != nil {
		return time.Time{}, err
	}
	return createdAt, nil
}

func findTags(s *bufio.Scanner) ([]string, error) {
	metadataStart := false
	tagsStart := false
	tags := make([]string, 0)
	for s.Scan() {
		line := s.Text()
		if strings.Contains(line, metadataBrace) {
			metadataStart = !metadataStart
			if !metadataStart {
				break
			}
			continue
		}
		if metadataStart {
			if tagsStart {
				if !strings.HasPrefix(line, tagBullet) {
					tagsStart = false
					continue
				}
				tag := strings.TrimPrefix(line, tagBullet)
				tags = append(tags, tag)
				continue
			}
			if strings.Contains(line, "tags:") {
				tagsStart = true
				continue
			}
		}
	}
	if s.Err() != nil {
		return nil, s.Err()
	}
	return tags, nil
}

func findCreatedAt(s *bufio.Scanner) (time.Time, error) {
	var err error
	var createdAt time.Time
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "created:") {
			timeString := strings.TrimSpace(strings.TrimPrefix(line, "created: "))
			createdAt, err = time.Parse("2006-01-02 15:04", timeString)
			if err != nil {
				return time.Time{}, err
			}
			break
		}
	}
	if s.Err() != nil {
		return time.Time{}, s.Err()
	}
	return createdAt, nil
}

func RemoveMetaData(ctx context.Context, r io.Reader) ([]byte, error) {
	scanner := bufio.NewScanner(r)
	f := make([]byte, 0)
	bWriter := bytes.NewBuffer(f)

	metadataStart := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, metadataBrace) {
			metadataStart = !metadataStart
			continue
		}
		if metadataStart {
			continue
		}
		if _, err := bWriter.WriteString(line + "\n"); err != nil {
			return nil, err
		}
	}
	if scanner.Err() != nil {
		return nil, scanner.Err()
	}
	return bWriter.Bytes(), nil
}
