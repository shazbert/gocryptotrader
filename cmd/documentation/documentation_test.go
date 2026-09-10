package main

import (
	"os"
	"path/filepath"
	"strings"
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
		{
			name:     "fence delimiter whitespace",
			input:    "```text   \n\tcontent  \n```   \n",
			expected: "```text\n\tcontent  \n```\n",
		},
		{
			name:     "list nested fence",
			input:    "1. Example\n\n    ```make\n    all:\n    \t@echo ok\n    ```\n",
			expected: "1. Example\n\n    ```make\n    all:\n    \t@echo ok\n    ```\n",
		},
		{
			name:     "blockquote nested fence",
			input:    "> ```make\n> all:\n> \t@echo ok\n> ```\n",
			expected: "> ```make\n> all:\n> \t@echo ok\n> ```\n",
		},
		{
			name:     "list marker inside fence",
			input:    "```text\n- ```\n\tcontent  \n```\n",
			expected: "```text\n- ```\n\tcontent  \n```\n",
		},
		{
			name:     "blockquote marker inside fence",
			input:    "```text\n> ```\n\tcontent  \n```\n",
			expected: "```text\n> ```\n\tcontent  \n```\n",
		},
		{
			name:     "ordered list marker inside fence",
			input:    "```text\n1. ```\n\tcontent  \n```\n",
			expected: "```text\n1. ```\n\tcontent  \n```\n",
		},
		{
			name:     "blockquote ends before fence",
			input:    "# Probe\n\n> ```text\n> content\n\n## Heading\n\n\n\nprose\twith tab  \n",
			expected: "# Probe\n\n> ```text\n> content\n\n## Heading\n\nprose    with tab\n",
		},
		{
			name:     "tab-indented list fence",
			input:    "- item\n\n\t```make\n\tall:\n\t\t@echo ok\n\t```\n",
			expected: "- item\n\n\t```make\n\tall:\n\t\t@echo ok\n\t```\n",
		},
		{
			name:     "empty list item converges",
			input:    "- \n    ```make\n    all:\n    \t@echo ok\n    ```\n",
			expected: "-\n    ```make\n    all:\n    \t@echo ok\n    ```\n",
		},
		{
			name:     "ordered list fence in blockquote",
			input:    "> 1. item\n>     ```make\n>     all:\n>     \t@echo ok\n>     ```\n",
			expected: "> 1. item\n>     ```make\n>     all:\n>     \t@echo ok\n>     ```\n",
		},
		{
			name:     "non-closing fence preserves whitespace",
			input:    "```text\n```not closing   \n\tcontent  \n```\n",
			expected: "```text\n```not closing   \n\tcontent  \n```\n",
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

func TestDonationHeading(t *testing.T) {
	t.Parallel()

	tmpl, err := template.ParseFiles(filepath.Join("sub_templates", "donations.tmpl"))
	require.NoError(t, err, "parsing donations template must not error")
	for _, name := range []string{"backtester", "exchanges subscription"} {
		var output strings.Builder
		err := tmpl.ExecuteTemplate(&output, "donations", GetDocumentationAttributes(name, nil))
		require.NoError(t, err, "rendering donations template must not error")
		assert.Contains(t, output.String(), "### Donations", "nested package donation heading should preserve hierarchy")
	}
}

func TestGetContributorList(t *testing.T) {
	t.Parallel()

	c, err := GetContributorList(t.Context(), DefaultRepo, true)
	require.NoError(t, err, "GetContributorList must not error")
	require.NotEmpty(t, c, "GetContributorList must not return empty list")
}
