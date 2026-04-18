package extractor

import (
	"strings"
	"testing"
)

func TestExtractPrefersArticleContent(t *testing.T) {
	html := `
	<html>
	  <head><title>Example Title</title></head>
	  <body>
	    <aside class="sidebar">short nav links</aside>
	    <article class="post-content">
	      <h1>Main Article</h1>
	      <p>This is the primary article content with enough punctuation, sentences, and detail to score highly.</p>
	      <p>It should beat the sidebar because it contains real paragraphs and article markers.</p>
	    </article>
	  </body>
	</html>`

	result := Extract(html)

	if result.Title != "Example Title" {
		t.Fatalf("unexpected title: %s", result.Title)
	}
	if !strings.Contains(result.Text, "primary article content") {
		t.Fatalf("expected article body in extraction, got: %s", result.Text)
	}
	if strings.Contains(result.Text, "short nav links") {
		t.Fatalf("did not expect sidebar text in extraction: %s", result.Text)
	}
	if !strings.Contains(result.ContentHTML, "<article") {
		t.Fatalf("expected extracted html to contain article markup, got: %s", result.ContentHTML)
	}
}

func TestHTMLToMarkdownRendersStructuredContent(t *testing.T) {
	input := `
	<h2>Heading</h2>
	<p>Hello <strong>world</strong> and <a href="https://example.com">friends</a>.</p>
	<ul><li>One</li><li>Two</li></ul>
	<pre><code>const x = 1;</code></pre>`

	output := HTMLToMarkdown(input)

	expectedParts := []string{
		"## Heading",
		"Hello **world** and [friends](https://example.com).",
		"- One",
		"- Two",
		"```",
		"const x = 1;",
	}
	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Fatalf("expected markdown to contain %q, got: %s", part, output)
		}
	}
}
