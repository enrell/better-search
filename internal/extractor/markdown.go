package extractor

import (
	"fmt"
	"regexp"
	"strings"
)

func HTMLToMarkdown(htmlStr string) string {
	md := htmlStr

	md = processPreCode(md)
	md = processHeaders(md)
	md = processLists(md)
	md = processLinks(md)
	md = processImages(md)
	md = processBoldItalic(md)
	md = processBlockquotes(md)
	md = processParagraphs(md)
	md = processLineBreaks(md)
	md = cleanWhitespace(md)

	return md
}

func processPreCode(html string) string {
	preRe := regexp.MustCompile(`(?is)<pre[^>]*>(.*?)</pre>`)
	html = preRe.ReplaceAllStringFunc(html, func(match string) string {
		inner := preRe.FindStringSubmatch(match)
		if len(inner) > 1 {
			content := stripTags(inner[1])
			return fmt.Sprintf("\n```\n%s\n```\n", content)
		}
		return match
	})

	codeRe := regexp.MustCompile(`(?is)<code[^>]*>(.*?)</code>`)
	html = codeRe.ReplaceAllStringFunc(html, func(match string) string {
		inner := codeRe.FindStringSubmatch(match)
		if len(inner) > 1 {
			content := stripTags(inner[1])
			if strings.Contains(content, "\n") {
				return fmt.Sprintf("\n```\n%s\n```\n", content)
			}
			return fmt.Sprintf("`%s`", content)
		}
		return match
	})

	return html
}

func processHeaders(html string) string {
	for level := 6; level >= 1; level-- {
		prefix := strings.Repeat("#", level)
		re := regexp.MustCompile(fmt.Sprintf(`(?is)<h%d[^>]*>(.*?)</h%d>`, level, level))
		html = re.ReplaceAllStringFunc(html, func(match string) string {
			inner := re.FindStringSubmatch(match)
			if len(inner) > 1 {
				content := stripTags(inner[1])
				return fmt.Sprintf("\n%s %s\n\n", prefix, content)
			}
			return match
		})
	}
	return html
}

func processLists(html string) string {
	html = regexp.MustCompile(`(?is)<ul[^>]*>`).ReplaceAllString(html, "\n")
	html = regexp.MustCompile(`(?is)</ul>`).ReplaceAllString(html, "\n")
	html = regexp.MustCompile(`(?is)<ol[^>]*>`).ReplaceAllString(html, "\n")
	html = regexp.MustCompile(`(?is)</ol>`).ReplaceAllString(html, "\n")

	html = regexp.MustCompile(`(?is)<li[^>]*>(.*?)</li>`).ReplaceAllStringFunc(html, func(match string) string {
		inner := regexp.MustCompile(`(?is)<li[^>]*>(.*?)</li>`).FindStringSubmatch(match)
		if len(inner) > 1 {
			content := strings.TrimSpace(stripTags(inner[1]))
			return fmt.Sprintf("- %s\n", content)
		}
		return match
	})

	return html
}

func processLinks(html string) string {
	return regexp.MustCompile(`(?is)<a[^>]*href=["']([^"']*)["'][^>]*>(.*?)</a>`).ReplaceAllStringFunc(html, func(match string) string {
		parts := regexp.MustCompile(`(?is)<a[^>]*href=["']([^"']*)["'][^>]*>(.*?)</a>`).FindStringSubmatch(match)
		if len(parts) > 2 {
			url := parts[1]
			text := stripTags(parts[2])
			return fmt.Sprintf("[%s](%s)", text, url)
		}
		return match
	})
}

func processImages(html string) string {
	html = regexp.MustCompile(`(?is)<img[^>]*src=["']([^"']*)["'][^>]*alt=["']([^"']*)["'][^>]*/?>`).ReplaceAllStringFunc(html, func(match string) string {
		parts := regexp.MustCompile(`(?is)<img[^>]*src=["']([^"']*)["'][^>]*alt=["']([^"']*)["'][^>]*/?>`).FindStringSubmatch(match)
		if len(parts) > 2 {
			return fmt.Sprintf("![%s](%s)", parts[2], parts[1])
		}
		return match
	})

	html = regexp.MustCompile(`(?is)<img[^>]*src=["']([^"']*)["'][^>]*/?>`).ReplaceAllStringFunc(html, func(match string) string {
		parts := regexp.MustCompile(`(?is)<img[^>]*src=["']([^"']*)["'][^>]*/?>`).FindStringSubmatch(match)
		if len(parts) > 1 {
			return fmt.Sprintf("![](%s)", parts[1])
		}
		return match
	})

	return html
}

func processBoldItalic(html string) string {
	html = regexp.MustCompile(`(?is)<strong[^>]*>(.*?)</strong>`).ReplaceAllStringFunc(html, func(match string) string {
		inner := regexp.MustCompile(`(?is)<strong[^>]*>(.*?)</strong>`).FindStringSubmatch(match)
		if len(inner) > 1 {
			return fmt.Sprintf("**%s**", stripTags(inner[1]))
		}
		return match
	})

	html = regexp.MustCompile(`(?is)<b[^>]*>(.*?)</b>`).ReplaceAllStringFunc(html, func(match string) string {
		inner := regexp.MustCompile(`(?is)<b[^>]*>(.*?)</b>`).FindStringSubmatch(match)
		if len(inner) > 1 {
			return fmt.Sprintf("**%s**", stripTags(inner[1]))
		}
		return match
	})

	html = regexp.MustCompile(`(?is)<em[^>]*>(.*?)</em>`).ReplaceAllStringFunc(html, func(match string) string {
		inner := regexp.MustCompile(`(?is)<em[^>]*>(.*?)</em>`).FindStringSubmatch(match)
		if len(inner) > 1 {
			return fmt.Sprintf("*%s*", stripTags(inner[1]))
		}
		return match
	})

	html = regexp.MustCompile(`(?is)<i[^>]*>(.*?)</i>`).ReplaceAllStringFunc(html, func(match string) string {
		inner := regexp.MustCompile(`(?is)<i[^>]*>(.*?)</i>`).FindStringSubmatch(match)
		if len(inner) > 1 {
			return fmt.Sprintf("*%s*", stripTags(inner[1]))
		}
		return match
	})

	return html
}

func processBlockquotes(html string) string {
	return regexp.MustCompile(`(?is)<blockquote[^>]*>(.*?)</blockquote>`).ReplaceAllStringFunc(html, func(match string) string {
		inner := regexp.MustCompile(`(?is)<blockquote[^>]*>(.*?)</blockquote>`).FindStringSubmatch(match)
		if len(inner) > 1 {
			content := stripTags(inner[1])
			lines := strings.Split(content, "\n")
			for i, line := range lines {
				lines[i] = "> " + line
			}
			return strings.Join(lines, "\n")
		}
		return match
	})
}

func processParagraphs(html string) string {
	return regexp.MustCompile(`(?is)<p[^>]*>(.*?)</p>`).ReplaceAllStringFunc(html, func(match string) string {
		inner := regexp.MustCompile(`(?is)<p[^>]*>(.*?)</p>`).FindStringSubmatch(match)
		if len(inner) > 1 {
			content := strings.TrimSpace(stripTags(inner[1]))
			if content != "" {
				return content + "\n\n"
			}
		}
		return match
	})
}

func processLineBreaks(html string) string {
	return regexp.MustCompile(`(?is)<br\s*/?>`).ReplaceAllString(html, "\n")
}

func cleanWhitespace(markdown string) string {
	markdown = regexp.MustCompile(`\n{3,}`).ReplaceAllString(markdown, "\n\n")
	return strings.TrimSpace(markdown)
}

func stripTags(html string) string {
	result := regexp.MustCompile(`<[^>]+>`).ReplaceAllString(html, "")
	result = strings.ReplaceAll(result, "&nbsp;", " ")
	result = strings.ReplaceAll(result, "&amp;", "&")
	result = strings.ReplaceAll(result, "&lt;", "<")
	result = strings.ReplaceAll(result, "&gt;", ">")
	result = strings.ReplaceAll(result, "&quot;", "\"")
	result = strings.ReplaceAll(result, "&#39;", "'")
	result = strings.ReplaceAll(result, "&mdash;", "—")
	result = strings.ReplaceAll(result, "&ndash;", "–")
	result = strings.ReplaceAll(result, "&hellip;", "…")
	return result
}
