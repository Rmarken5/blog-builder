package build

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"html/template"
	"io"
	"log"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/rmarken5/blog-builder/tool/logic/aws"
	"golang.org/x/sync/errgroup"
)

//go:embed templates/html-metadata-tags.html
var tagTemplate string

//go:embed templates/html-metadata-created-at.html
var createdAtTemplate string

var _ HTMLHandler = HandleHTML{}

const (
	HTMLFileExtension = ".html"
	injectAfter       = "</h1>"
)

type (
	HTMLHandler interface {
		GetHTMLFilesFromBuildPath(ctx context.Context, path string) ([]ReaderWithPath, error)
		ConvertMDToHTML(ctx context.Context, r io.Reader) ([]byte, error)
		ConvertMdLinksToHtml(r io.Reader) ([]byte, error)
		WriteHTML(ctx context.Context, w io.Writer, data []byte) error
		CreateFileFromMDPath(ctx context.Context, path string) (*os.File, error)
		CreateBuildDirectoryForPath(ctx context.Context, path string) (string, error)
		UploadHTML(ctx context.Context, upload map[string]io.Reader) error
	}
	HandleHTML struct {
		fileExtension   string
		markdownPath    string
		buildOutputPath string
		cssHandler      CSSHandler
		awsClient       aws.S3Client
	}
)

func (h HandleHTML) UploadHTML(ctx context.Context, upload map[string]io.Reader) error {
	errG, ctx := errgroup.WithContext(ctx)
	for k, r := range upload {
		errG.Go(func() error {
			return h.awsClient.WriteHTMLToBucket(ctx, k, r)
		})
	}
	return errG.Wait()
}

func NewHandleHTML(markdownDirectory string, buildOutputPath string, awsClient aws.S3Client) *HandleHTML {
	return &HandleHTML{
		markdownPath:    markdownDirectory,
		fileExtension:   HTMLFileExtension,
		buildOutputPath: buildOutputPath,
		awsClient:       awsClient,
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

// ConvertMdLinksToHtml reads from an io.Reader and converts local .md anchor links to .html
func (h HandleHTML) ConvertMdLinksToHtml(r io.Reader) ([]byte, error) {
	// Read all content from the reader
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// Regex to match anchor tags with href attributes
	// This matches: <a href="..." ...> where href contains local paths
	anchorRegex := regexp.MustCompile(`<a\s+([^>]*\s+)?href="([^"]+)"([^>]*)>`)

	// Replace function to process each match
	result := anchorRegex.ReplaceAllFunc(content, func(match []byte) []byte {
		// Extract the href value
		hrefRegex := regexp.MustCompile(`href="([^"]+)"`)
		hrefMatch := hrefRegex.FindSubmatch(match)

		if len(hrefMatch) < 2 {
			return match
		}

		href := string(hrefMatch[1])

		// Check if it's a local link (not starting with http://, https://, //, etc.)
		if isLocalLink(href) && bytes.HasSuffix([]byte(href), []byte(".md")) {
			// Replace .md with .html
			newHref := href[:len(href)-3] + ".html"
			return bytes.Replace(match, []byte(href), []byte(newHref), 1)
		}

		return match
	})

	return result, nil
}

// isLocalLink checks if a URL is a local link
func isLocalLink(href string) bool {
	// Not local if it starts with a protocol
	if len(href) >= 7 && (href[:7] == "http://" || href[:8] == "https://") {
		return false
	}
	if len(href) >= 2 && href[:2] == "//" {
		return false
	}
	if len(href) >= 7 && href[:7] == "mailto:" {
		return false
	}
	if len(href) >= 4 && href[:4] == "ftp:" {
		return false
	}
	// Check for fragment-only links (like #section)
	if len(href) > 0 && href[0] == '#' {
		return false
	}

	return true
}

type Metadata struct {
	Tags      []string
	CreatedAt string
}

func InjectMetadataHeader(ctx context.Context, r io.Reader, metadata Metadata) ([]byte, error) {
	bufWritter := bytes.NewBuffer([]byte{})
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		b := scanner.Bytes()
		bufWritter.Write(append(b, []byte("\n")...))
		if strings.Contains(string(b), injectAfter) {
			t, err := template.New("").Parse(createdAtTemplate)
			if err != nil {
				return nil, err
			}
			err = t.ExecuteTemplate(bufWritter, "", metadata)
			if err != nil {
				return nil, err
			}
			t, err = template.New("").Parse(tagTemplate)
			if err != nil {
				return nil, err
			}
			err = t.ExecuteTemplate(bufWritter, "", metadata)
			if err != nil {
				return nil, err
			}
		}
	}
	return bufWritter.Bytes(), nil
}
