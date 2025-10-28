package main

import (
	"context"
	"log/slog"

	"github.com/rmarken5/blog-builder/tool/logic"
)

func main() {
	ctx := context.Background()
	markdownDir := "markdown"
	outputDir := "build"
	cssDirectory := "css"
	htmlHandler := logic.NewHandleHTML(markdownDir, outputDir)
	cssHandler := logic.NewHandleCSS(cssDirectory, outputDir+"/css", ".css")
	mdHandler := logic.NewHandleMarkdown()
	payloadBuilder := logic.NewPayloadBuilder(htmlHandler, cssHandler, mdHandler)

	err := payloadBuilder.BuildPayload(ctx, markdownDir, outputDir)
	if err != nil {
		slog.Error("error building html from markdown")
	}
}
