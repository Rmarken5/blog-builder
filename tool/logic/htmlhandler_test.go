package logic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandleHTML_GetHTMLFilesFromBuildPath(t *testing.T) {
	t.Run("check files", func(t *testing.T) {
		path, err := HandleHTML{fileExtension: ".html"}.GetHTMLFilesFromBuildPath("./test-files")
		assert.NoError(t, err)
		assert.Len(t, path, 1)
	})
}
