package extractor

import (
	"bytes"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

type markdownRenderer struct {
	builder bytes.Buffer
}

func HTMLToMarkdown(htmlStr string) string {
	if strings.TrimSpace(htmlStr) == "" {
		return ""
	}

	doc, err := html.Parse(strings.NewReader("<html><body>" + htmlStr + "</body></html>"))
	if err != nil {
		return strings.TrimSpace(stripTags(htmlStr))
	}

	renderer := &markdownRenderer{}
	body := findFirstNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "body"
	})
	if body == nil {
		return strings.TrimSpace(stripTags(htmlStr))
	}

	renderer.renderChildren(body, 0)
	return cleanWhitespace(renderer.builder.String())
}

func (r *markdownRenderer) renderChildren(node *html.Node, listDepth int) {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		r.renderNode(child, listDepth)
	}
}

func (r *markdownRenderer) renderNode(node *html.Node, listDepth int) {
	switch node.Type {
	case html.TextNode:
		text := normalizeText(node.Data)
		if text != "" {
			r.builder.WriteString(text)
		}
	case html.ElementNode:
		switch node.Data {
		case "h1", "h2", "h3", "h4", "h5", "h6":
			level := int(node.Data[1] - '0')
			r.block(strings.Repeat("#", level) + " " + strings.TrimSpace(r.inline(node)))
		case "p":
			content := strings.TrimSpace(r.inline(node))
			if content != "" {
				r.block(content)
			}
		case "br":
			r.builder.WriteByte('\n')
		case "blockquote":
			content := strings.TrimSpace(r.inline(node))
			if content == "" {
				return
			}
			lines := strings.Split(content, "\n")
			for i, line := range lines {
				lines[i] = "> " + strings.TrimSpace(line)
			}
			r.block(strings.Join(lines, "\n"))
		case "pre":
			content := strings.TrimSpace(r.inline(node))
			if content != "" {
				r.block("```\n" + content + "\n```")
			}
		case "ul":
			r.renderList(node, listDepth, false)
			r.builder.WriteByte('\n')
		case "ol":
			r.renderList(node, listDepth, true)
			r.builder.WriteByte('\n')
		case "hr":
			r.block("---")
		default:
			r.renderChildren(node, listDepth)
		}
	}
}

func (r *markdownRenderer) renderList(node *html.Node, listDepth int, ordered bool) {
	index := 1
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode || child.Data != "li" {
			continue
		}

		prefix := "- "
		if ordered {
			prefix = strconv.Itoa(index) + ". "
		}
		indent := strings.Repeat("  ", listDepth)
		content, nested := renderListItem(child, listDepth+1)
		r.builder.WriteString(indent)
		r.builder.WriteString(prefix)
		r.builder.WriteString(strings.TrimSpace(content))
		r.builder.WriteByte('\n')
		if nested != "" {
			r.builder.WriteString(nested)
		}
		index++
	}
}

func renderListItem(node *html.Node, listDepth int) (string, string) {
	var inlineParts []string
	var nestedBuilder strings.Builder

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && (child.Data == "ul" || child.Data == "ol") {
			renderer := &markdownRenderer{}
			renderer.renderList(child, listDepth, child.Data == "ol")
			nestedBuilder.WriteString(renderer.builder.String())
			continue
		}
		inlineParts = append(inlineParts, renderInlineNode(child))
	}

	return strings.TrimSpace(strings.Join(inlineParts, "")), nestedBuilder.String()
}

func (r *markdownRenderer) inline(node *html.Node) string {
	return renderInlineNode(node)
}

func renderInlineNode(node *html.Node) string {
	switch node.Type {
	case html.TextNode:
		return normalizeText(node.Data)
	case html.ElementNode:
		switch node.Data {
		case "strong", "b":
			return "**" + strings.TrimSpace(renderInlineChildren(node)) + "**"
		case "em", "i":
			return "*" + strings.TrimSpace(renderInlineChildren(node)) + "*"
		case "code":
			return "`" + strings.TrimSpace(renderInlineChildren(node)) + "`"
		case "a":
			text := strings.TrimSpace(renderInlineChildren(node))
			href := strings.TrimSpace(getAttr(node, "href"))
			if href == "" {
				return text
			}
			if text == "" {
				text = href
			}
			return "[" + text + "](" + href + ")"
		case "img":
			src := strings.TrimSpace(getAttr(node, "src"))
			if src == "" {
				return ""
			}
			alt := strings.TrimSpace(getAttr(node, "alt"))
			return "![" + alt + "](" + src + ")"
		case "br":
			return "\n"
		default:
			return renderInlineChildren(node)
		}
	}

	return ""
}

func renderInlineChildren(node *html.Node) string {
	var builder strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		builder.WriteString(renderInlineNode(child))
	}
	return builder.String()
}

func normalizeText(text string) string {
	leadingSpace := len(text) > 0 && isSpace(text[0])
	trailingSpace := len(text) > 0 && isSpace(text[len(text)-1])

	text = html.UnescapeString(text)
	text = strings.ReplaceAll(text, "\u00a0", " ")
	text = collapseWhitespace(text)
	if text == "" {
		return ""
	}
	if leadingSpace {
		text = " " + text
	}
	if trailingSpace {
		text += " "
	}
	return text
}

func collapseWhitespace(text string) string {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}
	return strings.Join(fields, " ")
}

func (r *markdownRenderer) block(content string) {
	if strings.TrimSpace(content) == "" {
		return
	}
	if r.builder.Len() > 0 && !strings.HasSuffix(r.builder.String(), "\n\n") {
		if strings.HasSuffix(r.builder.String(), "\n") {
			r.builder.WriteByte('\n')
		} else {
			r.builder.WriteString("\n\n")
		}
	}
	r.builder.WriteString(content)
	r.builder.WriteString("\n\n")
}

func cleanWhitespace(markdown string) string {
	lines := strings.Split(markdown, "\n")
	cleaned := make([]string, 0, len(lines))
	blankCount := 0

	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) == "" {
			blankCount++
			if blankCount > 1 {
				continue
			}
		} else {
			blankCount = 0
		}
		cleaned = append(cleaned, line)
	}

	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

func stripTags(htmlStr string) string {
	var builder strings.Builder
	tokenizer := html.NewTokenizer(strings.NewReader(htmlStr))

	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			return html.UnescapeString(builder.String())
		case html.TextToken:
			builder.WriteString(tokenizer.Token().Data)
		}
	}
}

func isSpace(b byte) bool {
	switch b {
	case ' ', '\n', '\r', '\t':
		return true
	default:
		return false
	}
}
