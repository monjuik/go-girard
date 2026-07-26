package app

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

var noteMarkdown = goldmark.New(
	goldmark.WithExtensions(
		extension.Strikethrough,
		extension.TaskList,
	),
)

func renderMarkdown(source string) (template.HTML, error) {
	var output bytes.Buffer
	if err := noteMarkdown.Convert([]byte(source), &output); err != nil {
		return "", fmt.Errorf("render markdown: %w", err)
	}

	return template.HTML(output.String()), nil
}
