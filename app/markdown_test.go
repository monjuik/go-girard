package app

import (
	"strings"
	"testing"
)

func TestRenderMarkdown(t *testing.T) {
	source := `# Responsibilities

  [Website](https://example.com)

  **Bold**, *italic* and ~~obsolete~~.

  - Regular item
  - [ ] Pending
  - [x] Completed
  `

	rendered, err := renderMarkdown(source)
	if err != nil {
		t.Fatalf("renderMarkdown() error = %v", err)
	}

	html := string(rendered)
	for _, want := range []string{
		"<h1>Responsibilities</h1>",
		`href="https://example.com"`,
		"<strong>Bold</strong>",
		"<em>italic</em>",
		"<del>obsolete</del>",
		"<ul>",
		`type="checkbox"`,
		`checked=""`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered Markdown does not contain %q:\n%s", want, html)
		}
	}
}

func TestRenderMarkdownDoesNotRenderUnsafeHTML(t *testing.T) {
	source := `<script>alert("unsafe")</script>

  [Unsafe](javascript:alert("unsafe"))
  `

	rendered, err := renderMarkdown(source)
	if err != nil {
		t.Fatalf("renderMarkdown() error = %v", err)
	}

	html := string(rendered)
	for _, unsafe := range []string{"<script", "javascript:"} {
		if strings.Contains(html, unsafe) {
			t.Errorf("rendered Markdown contains unsafe value %q:\n%s", unsafe, html)
		}
	}
}
