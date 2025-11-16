package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rmarken5/blog-builder/tool/logic/aws"
	"github.com/rmarken5/blog-builder/tool/logic/build"
)

var bucketName = flag.String("bucket-name", "", "name of s3 bucket")
var region = flag.String("region", "us-east-2", "name of s3 region")
var markdownDir = flag.String("markdown-directory", "markdown", "path to markdown content directory")
var cssDirectory = flag.String("css-directory", "css", "path to css content directory")
var outputDir = flag.String("output-directory", "build", "path to output directory")
var withoutBuildOutput = flag.Bool("disable-local-output", false, "setting disable-local-output will upload files directly without writing to local build directory")
var disableUpload = flag.Bool("disable-upload", false, "setting the disable-upload flag will run the build without pushing the build to s3")

func main() {

	flag.Parse()
	// shouldBuildLocal := !*withoutBuildOutput
	log.Println("WithoutUpload: ", *disableUpload)
	uploadDisabled := *disableUpload

	if !uploadDisabled && (bucketName == nil || *bucketName == "") {
		log.Printf("-bucket-name is required")
		os.Exit(99)
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(*region))
	if err != nil {
		log.Fatal(err)
	}

	// Create an Amazon S3 service client
	client := s3.NewFromConfig(cfg)
	ctx := context.Background()
	var s3Client aws.S3Client
	s3Impl := aws.New(client, *bucketName)
	s3Client = s3Impl
	if uploadDisabled {
		s3Client = &aws.NoOp{
			S3Client: s3Impl,
		}
	}

	htmlHandler := build.NewHandleHTML(*markdownDir, *outputDir, s3Client)
	cssHandler := build.NewHandleCSS(*cssDirectory, *outputDir+"/css", ".css", s3Client)
	mdHandler := build.NewHandleMarkdown()
	payloadBuilder := build.NewPayloadBuilder(htmlHandler, cssHandler, mdHandler, s3Client)

	err = payloadBuilder.BuildPayload(ctx, *markdownDir, *outputDir)
	if err != nil {
		slog.Error("error building html from markdown")
	}

}
