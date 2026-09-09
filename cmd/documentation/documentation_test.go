package main

import (
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunTemplateNormalizesMarkdown(t *testing.T) {
	t.Parallel()

	tmpl := template.Must(template.New("documentation").Parse("{{define \"documentation\"}}# Heading\r\n\r\n\r\nbefore\tafter  \r\n\r\n{{end}}"))
	outputPath := filepath.Join(t.TempDir(), "README.md")
	err := runTemplate(DocumentationDetails{Tmpl: tmpl}, outputPath, "documentation")
	require.NoError(t, err, "runTemplate must not error")

	contents, err := os.ReadFile(outputPath)
	require.NoError(t, err, "reading generated documentation must not error")
	assert.Equal(t, "# Heading\n\nbefore    after\n", string(contents), "runTemplate should normalize generated Markdown")
}

func TestGetContributorList(t *testing.T) {
	t.Parallel()

	c, err := GetContributorList(t.Context(), DefaultRepo, true)
	require.NoError(t, err, "GetContributorList must not error")
	require.NotEmpty(t, c, "GetContributorList must not return empty list")
}
