package build

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rmarken5/blog-builder/tool/logic/aws"
)

type (
	PayloadBuilder interface {
		BuildPayload(ctx context.Context, inputPath, payloadPath string) error
	}
	BuildPayload struct {
		htmlHandler     HTMLHandler
		cssHandler      CSSHandler
		markdownHandler MarkdownHandler
		s3Client        aws.S3Client
	}
)

func NewPayloadBuilder(htmlHandler HTMLHandler, cssHandler CSSHandler, markdownHandler MarkdownHandler, s3Client aws.S3Client) *BuildPayload {
	return &BuildPayload{
		htmlHandler:     htmlHandler,
		cssHandler:      cssHandler,
		markdownHandler: markdownHandler,
		s3Client:        s3Client,
	}
}

func (b BuildPayload) BuildToS3(ctx context.Context, inputPath, payloadPath string) error {
	uploadedFiles := make([]string, 0)
	rHashes, err := b.s3Client.GetBucketHashes(ctx)
	if err != nil {
		slog.Error("error calculating hash from s3", "error", err)

	}
	slog.Info("remote hashes", "hashes", rHashes)

	localHashes := make(map[string]string, 0)
	log.Println("creating build directory")

	cssFiles, err := b.cssHandler.GetCSSFilesFromSource(ctx)
	if err != nil {
		slog.Error("error getting css files from css directory", "error", err)
		return err
	}

	for _, cssFile := range cssFiles {
		lKey := strings.TrimPrefix(cssFile.Path, payloadPath+"/")
		minifiedBytes, err := b.cssHandler.MinifyCSS(ctx, cssFile.Reader)
		if err != nil {
			slog.Error("error minifying css file", "error", err)
			return err
		}

		hash, err := calcMD5(bytes.NewReader(minifiedBytes))
		if err != nil {
			slog.Error("error calculating hash", "error", err)
		}

		localHashes[lKey] = hash
		if shouldUpload(rHashes, lKey, hash) {
			slog.Info("No matching hash, writing file to s3", "file", lKey)
			uploadedFiles = append(uploadedFiles, lKey)
			err = b.s3Client.WriteFileToBucket(ctx, lKey, "text/css", bytes.NewReader(minifiedBytes))
			if err != nil {
				slog.Error("error writing to s3", "key", lKey, "error", err)
			}
		}
	}

	log.Printf("reading markdown from %s", inputPath)
	markdownFiles, err := b.markdownHandler.GetMarkdownFilesFromPath(ctx, inputPath)
	if err != nil {
		slog.Error("error reading markdown directory", "error", err)
		return err
	}
	for _, mdFile := range markdownFiles {
		mdBytes := bytes.NewBuffer([]byte{})
		_, err := io.Copy(mdBytes, mdFile.Reader)
		if err != nil {
			slog.Error("error copying bytes to buffer for mdfile", "error", err)
		}
		mdFile.Reader.Close()

		tags, err := GetTags(ctx, bytes.NewReader(mdBytes.Bytes()), findTags)
		if err != nil {
			slog.Error("error getting tags from markdown file", "error", err)
			return err
		}
		createdAtDate, err := GetCreatedAtDate(ctx, bytes.NewReader(mdBytes.Bytes()), findCreatedAt)
		if err != nil {
			slog.Error("error getting createdAt date from md", "error", err)
		}
		mdStripped, err := RemoveMetaData(ctx, bytes.NewReader(mdBytes.Bytes()))
		if err != nil {
			slog.Error("error removing metadata from md", "error", err)
		}
		htmlBytes, err := b.htmlHandler.ConvertMDToHTML(ctx, bytes.NewReader(mdStripped))
		if err != nil {
			slog.Error("error converting md to html", "error", err)
			return err
		}
		htmlBytes, err = InjectMetadataHeader(ctx, bytes.NewReader(htmlBytes), Metadata{
			Tags:      tags,
			CreatedAt: createdAtDate.Format(time.RFC850),
		})
		if err != nil {
			slog.Error("error injecting metadata into html", "error", err)
		}
		for _, cssFile := range cssFiles {
			cssPath := strings.TrimPrefix(cssFile.Path, payloadPath+"/")
			count := strings.Count(mdFile.Path, "/")
			cssPath = strings.Repeat("../", count-1) + cssPath

			htmlBytes, err = b.cssHandler.InjectCSSIntoHTML(ctx, bytes.NewReader(htmlBytes), cssPath)
			if err != nil {
				slog.Error("error injecting css into html", "error", err)
				return err
			}
		}

		htmlBytes, err = b.htmlHandler.ConvertMdLinksToHtml(bytes.NewReader(htmlBytes))
		if err != nil {
			slog.Error("error converting md to html", "error", err)
			return err
		}

		hash, err := calcMD5(bytes.NewReader(htmlBytes))
		if err != nil {
			slog.Error("error calculating hash for html", "path", mdFile.Path, "error", err)
			return err
		}

		lKey := strings.Replace(strings.TrimPrefix(mdFile.Path, inputPath+"/"), markdownFileExtension, HTMLFileExtension, -1)
		if shouldUpload(rHashes, lKey, hash) {
			slog.Info("No matching hash, writing file to s3", "file", lKey)
			uploadedFiles = append(uploadedFiles, lKey)
			err = b.s3Client.WriteFileToBucket(ctx, lKey, "text/html", bytes.NewReader(htmlBytes))
			if err != nil {
				slog.Error("error writing to s3", "key", lKey, "error", err)
			}
		}
		localHashes[lKey] = hash
	}

	log.Printf("files written to s3: %v", uploadedFiles)

	return nil
}

func (b BuildPayload) BuildPayload(ctx context.Context, inputPath, payloadPath string) error {
	rHashes, err := b.s3Client.GetBucketHashes(ctx)
	if err != nil {
		slog.Error("error calculating hash from s3", "error", err)

	}
	slog.Info("remote hashes", "hashes", rHashes)

	localHashes := make(map[string]string)
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

	cssFiles, err := b.cssHandler.GetCSSFilesFromSource(ctx)
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

		cssBuildFile, err := b.cssHandler.CreateBuildFileFromCSSSource(ctx, cssFile.Path)
		if err != nil {
			slog.Error("error creating css file", "error", err)
			return err
		}
		_, err = cssBuildFile.Write(minifiedBytes)
		if err != nil {
			slog.Error("error writing minified bytes to build output", "error", err)
			return err
		}
		cssBuildFile.Close()

		hash, err := calcMD5(bytes.NewReader(minifiedBytes))
		if err != nil {
			slog.Error("error calculating hash", "error", err)
		}

		lKey := strings.TrimPrefix(cssFile.Path, payloadPath+"/")
		if shouldUpload(rHashes, lKey, hash) {
			slog.Info("No matching hash, writing file to s3", "file", lKey)
			err = b.s3Client.WriteFileToBucket(ctx, lKey, "text/css", bytes.NewReader(minifiedBytes))
			if err != nil {
				slog.Error("error writing to s3", "key", lKey, "error", err)
			}
		}
		localHashes[lKey] = hash
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
		mdBytes := bytes.NewBuffer([]byte{})
		_, err := io.Copy(mdBytes, mdFile.Reader)
		if err != nil {
			slog.Error("error copying bytes to buffer for mdfile", "error", err)
		}
		mdFile.Reader.Close()

		tags, err := GetTags(ctx, bytes.NewReader(mdBytes.Bytes()), findTags)
		if err != nil {
			slog.Error("error getting tags from markdown file", "error", err)
			return err
		}
		createdAtDate, err := GetCreatedAtDate(ctx, bytes.NewReader(mdBytes.Bytes()), findCreatedAt)
		if err != nil {
			slog.Error("error getting createdAt date from md", "error", err)
		}
		mdStripped, err := RemoveMetaData(ctx, bytes.NewReader(mdBytes.Bytes()))
		if err != nil {
			slog.Error("error removing metadata from md", "error", err)
		}
		htmlBytes, err := b.htmlHandler.ConvertMDToHTML(ctx, bytes.NewReader(mdStripped))
		if err != nil {
			slog.Error("error converting md to html", "error", err)
			return err
		}
		htmlBytes, err = InjectMetadataHeader(ctx, bytes.NewReader(htmlBytes), Metadata{
			Tags:      tags,
			CreatedAt: createdAtDate.Format(time.RFC850),
		})
		if err != nil {
			slog.Error("error injecting metadata into html", "error", err)
		}

		for _, css := range cssBuildFiles {
			cssPath := strings.TrimPrefix(css.Path, payloadPath+"/")
			p := strings.Replace(mdFile.Path, inputPath, payloadPath, -1)
			log.Println("path, ", p)
			count := strings.Count(p, "/")
			cssPath = strings.Repeat("../", count-1) + cssPath

			htmlBytes, err = b.cssHandler.InjectCSSIntoHTML(ctx, bytes.NewReader(htmlBytes), cssPath)
			if err != nil {
				slog.Error("error injecting css into html", "error", err)
				return err
			}
		}

		htmlBytes, err = b.htmlHandler.ConvertMdLinksToHtml(bytes.NewReader(htmlBytes))
		if err != nil {
			slog.Error("error converting md to html", "error", err)
			return err
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

		hash, err := calcMD5(bytes.NewReader(htmlBytes))
		if err != nil {
			slog.Error("error calculating hash for html", "path", mdFile.Path, "error", err)
			return err
		}
		lKey := strings.Replace(strings.TrimPrefix(mdFile.Path, inputPath+"/"), markdownFileExtension, HTMLFileExtension, -1)
		if shouldUpload(rHashes, lKey, hash) {
			slog.Info("No matching hash, writing file to s3", "file", lKey)
			err = b.s3Client.WriteFileToBucket(ctx, lKey, "text/html", bytes.NewReader(htmlBytes))
			if err != nil {
				slog.Error("error writing to s3", "key", lKey, "error", err)
			}
		}
		localHashes[lKey] = hash
	}

	for _, css := range cssBuildFiles {
		css.Reader.Close()
	}

	slog.Info("local hashes", "hashes", localHashes)

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

func calcMD5(r io.Reader) (string, error) {
	hash := md5.New()
	if _, err := io.Copy(hash, r); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
func shouldUpload(rHashes map[string]string, lKey, lHash string) bool {
	if rHash, ok := rHashes[lKey]; !ok || lHash != rHash {
		return true
	}
	return false
}
