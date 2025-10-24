package main

import (
	"github.com/rmarken5/blog-builder/tool/logic"
	"log/slog"
)

func main() {
	markdownDir := "markdown"
	outputDir := "build"
	htmlBuilder := logic.New(outputDir)

	err := htmlBuilder.BuildHTMLFromMD(markdownDir)
	if err != nil {
		slog.Error("error building html from markdown")
	}
}
