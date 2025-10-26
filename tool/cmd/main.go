package main

import (
	"log/slog"

	"github.com/rmarken5/blog-builder/tool/logic"
)

func main() {
	markdownDir := "markdown"
	outputDir := "build"
	htmlHandler := logic.NewHandleHTML(outputDir)
	cssHandler := logic.NewHandleCSS(".css")
	mdHandler := logic.NewHandleMarkdown()
	payloadBuilder := logic.NewPayloadBuilder(htmlHandler, cssHandler, mdHandler)

	err := htmlHandler.BuildHTMLFromMD(markdownDir)
	if err != nil {
		slog.Error("error building html from markdown")
	}
}
