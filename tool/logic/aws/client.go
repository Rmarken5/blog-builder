package aws

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var (
	ErrUploadFile = errors.New("error uploading file to s3")
)

const (
	contentTypeTextCSS  = "text/css"
	contentTypeTextHTML = "text/html"
)

type (
	S3Client interface {
		GetBucketHashes(ctx context.Context) (map[string]string, error)
		WriteHTMLToBucket(ctx context.Context, key string, file io.Reader) error
		WriteCSSToBucket(ctx context.Context, key string, file io.Reader) error
	}
	Client struct {
		client *s3.Client
		bucket string
	}
	NoOp struct {
		S3Client
	}
)

func New(client *s3.Client, bucket string) *Client {
	return &Client{
		client: client,
		bucket: bucket,
	}
}

func (c Client) GetBucketHashes(ctx context.Context) (map[string]string, error) {
	hashes := make(map[string]string)
	objectList, err := c.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: aws.String(c.bucket)})
	if err != nil {
		slog.Error("error listing objects", "error", err)
		return nil, err
	}

	for _, object := range objectList.Contents {
		key := *object.Key
		if key[len(key)-1:] == "/" {
			continue
		}
		getObject, err := c.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(c.bucket), Key: object.Key})
		if err != nil {
			slog.Error("error getting object", "error", err, "bucket", c.bucket, "key", key)
			return nil, err
		}

		hash, err := calcMD5(getObject.Body)
		getObject.Body.Close()
		if err != nil {
			slog.Error("error generating hash for object", "error", err, "bucket", c.bucket, "key", key)
		}
		hashes[*object.Key] = hash
	}

	return hashes, nil
}

func calcMD5(r io.Reader) (string, error) {
	hash := md5.New()
	if _, err := io.Copy(hash, r); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (c Client) WriteHTMLToBucket(ctx context.Context, key string, file io.Reader) error {
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentTypeTextHTML),
	})
	if err != nil {
		slog.Error("error uploading file to s3", "filename", key, "error", err)
		return fmt.Errorf("error writing file %s to s3: %w - %w", key, err, ErrUploadFile)
	}

	return nil
}
func (c Client) WriteCSSToBucket(ctx context.Context, key string, file io.Reader) error {
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentTypeTextCSS),
	})
	if err != nil {
		slog.Error("error uploading file to s3", "filename", key, "error", err)
		return fmt.Errorf("error writing file %s to s3: %w - %w", key, err, ErrUploadFile)
	}

	return nil
}

func (n NoOp) WriteHTMLToBucket(ctx context.Context, key string, file io.Reader) error {
	slog.Info("NoOp Write for HTML", "key", key)
	return nil
}
func (n NoOp) WriteCSSToBucket(ctx context.Context, key string, file io.Reader) error {
	slog.Info("noop wirte for CSS", "key", key)
	return nil
}
