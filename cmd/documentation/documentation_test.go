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

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "prose whitespace",
			input:    "# Heading\r\n\r\n\r\nbefore\tafter  \r\n\r\n",
			expected: "# Heading\n\nbefore    after\n",
		},
		{
			name:     "fenced whitespace",
			input:    "```go\r\n\tfirst  \r\n\r\n\r\n\tsecond\r\n```\r\n",
			expected: "```go\n\tfirst  \n\n\n\tsecond\n```\n",
		},
		{
			name:     "tilde fence",
			input:    "~~~text\n\tcontent\n~~~~\n",
			expected: "~~~text\n\tcontent\n~~~~\n",
		},
		{
			name:     "fence-like content",
			input:    "````text\n```not a closing fence\n\tcontent\n````\n",
			expected: "````text\n```not a closing fence\n\tcontent\n````\n",
		},
		{
			name:     "indented fence",
			input:    "   ```text\n\tcontent  \n   ```\n",
			expected: "   ```text\n\tcontent  \n   ```\n",
		},
		{
			name:     "unmatched fence",
			input:    "```text\n\tcontent  \n\n\n",
			expected: "```text\n\tcontent  \n",
		},
		{
			name:     "code-indented fence",
			input:    "    ```text  \n\tprose  \n",
			expected: "    ```text\n    prose\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tmpl := template.Must(template.New("documentation").Parse("{{define \"documentation\"}}" + tt.input + "{{end}}"))
			outputPath := filepath.Join(t.TempDir(), "README.md")
			err := runTemplate(DocumentationDetails{Tmpl: tmpl}, outputPath, "documentation")
			require.NoError(t, err, "runTemplate must not error")

			contents, err := os.ReadFile(outputPath)
			require.NoError(t, err, "reading generated documentation must not error")
			assert.Equal(t, tt.expected, string(contents), "runTemplate should normalize generated Markdown")

			info, err := os.Stat(outputPath)
			require.NoError(t, err, "stat generated documentation must not error")
			assert.Zero(t, info.Mode().Perm()&0o111, "generated documentation should not be executable")
		})
	}
}

func TestGetContributorList(t *testing.T) {
	t.Parallel()

	c, err := GetContributorList(t.Context(), DefaultRepo, true)
	require.NoError(t, err, "GetContributorList must not error")
	require.NotEmpty(t, c, "GetContributorList must not return empty list")
}
