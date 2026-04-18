package extractor

import (
	"bytes"
	"math"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var removeTags = map[string]bool{
	"script": true, "style": true, "nav": true, "aside": true, "footer": true, "header": true,
	"form": true, "iframe": true, "noscript": true, "svg": true, "button": true, "input": true,
	"select": true, "textarea": true, "menu": true, "figure": true, "figcaption": true,
}

var boostClasses = []string{"content", "article", "post", "entry", "main", "text", "body"}
var penaltyClasses = []string{"comment", "sidebar", "footer", "header", "nav", "menu", "widget", "ad", "advertisement", "social", "share"}

type Metadata struct {
	Title    string `json:"title"`
	Author   string `json:"author"`
	Date     string `json:"date"`
	Language string `json:"language"`
	URL      string `json:"url"`
}

type ExtractionResult struct {
	Title    string
	Text     string
	Author   string
	Date     string
	Language string
	URL      string
}

func Extract(htmlStr string) ExtractionResult {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return ExtractionResult{}
	}

	removeUnwantedNodes(doc)
	meta := extractMetadata(doc)
	mainContent := findMainContent(doc)
	cleaned := cleanContent(mainContent)

	return ExtractionResult{
		Title:    meta.Title,
		Text:     cleaned,
		Author:   meta.Author,
		Date:     meta.Date,
		Language: meta.Language,
		URL:      meta.URL,
	}
}

func removeUnwantedNodes(n *html.Node) {
	var toRemove []*html.Node
	var walker func(*html.Node)
	walker = func(node *html.Node) {
		if node.Type == html.ElementNode && removeTags[node.Data] {
			toRemove = append(toRemove, node)
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walker(child)
		}
	}
	walker(n)

	for _, node := range toRemove {
		if node.Parent != nil {
			node.Parent.RemoveChild(node)
		}
	}
}

func extractMetadata(doc *html.Node) Metadata {
	return Metadata{
		Title:    extractTitle(doc),
		Author:   extractAuthor(doc),
		Date:     extractDate(doc),
		Language: extractLanguage(doc),
		URL:      extractURL(doc),
	}
}

func extractTitle(doc *html.Node) string {
	if meta := findFirstNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "meta" && getAttr(n, "property") == "og:title"
	}); meta != nil {
		if title := getAttr(meta, "content"); title != "" {
			return strings.TrimSpace(title)
		}
	}

	if titleNode := findFirstNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "title"
	}); titleNode != nil {
		if title := getTextContent(titleNode); title != "" {
			return strings.TrimSpace(title)
		}
	}

	if h1 := findFirstNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "h1"
	}); h1 != nil {
		if title := getTextContent(h1); title != "" {
			return strings.TrimSpace(title)
		}
	}

	return ""
}

func extractAuthor(doc *html.Node) string {
	if n := findFirstNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "meta" && getAttr(n, "name") == "author"
	}); n != nil {
		if author := getAttr(n, "content"); author != "" {
			return strings.TrimSpace(author)
		}
	}

	if authorNode := findFirstNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "a" && getAttr(n, "rel") == "author"
	}); authorNode != nil {
		if author := getTextContent(authorNode); author != "" {
			return strings.TrimSpace(author)
		}
	}

	return ""
}

func extractDate(doc *html.Node) string {
	if n := findFirstNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "meta" && getAttr(n, "property") == "article:published_time"
	}); n != nil {
		if date := getAttr(n, "content"); date != "" {
			return strings.TrimSpace(date)
		}
	}

	if timeNode := findFirstNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "time"
	}); timeNode != nil {
		date := getAttr(timeNode, "datetime")
		if date == "" {
			date = getTextContent(timeNode)
		}
		if date != "" {
			return strings.TrimSpace(date)
		}
	}

	return ""
}

func extractLanguage(doc *html.Node) string {
	if htmlNode := findFirstNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "html"
	}); htmlNode != nil {
		lang := getAttr(htmlNode, "lang")
		if lang == "" {
			lang = getAttr(htmlNode, "xml:lang")
		}
		return strings.TrimSpace(lang)
	}
	return ""
}

func extractURL(doc *html.Node) string {
	if n := findFirstNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "meta" && getAttr(n, "property") == "og:url"
	}); n != nil {
		if u := getAttr(n, "content"); u != "" {
			return strings.TrimSpace(u)
		}
	}

	if link := findFirstNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "link" && getAttr(n, "rel") == "canonical"
	}); link != nil {
		if href := getAttr(link, "href"); href != "" {
			return strings.TrimSpace(href)
		}
	}

	return ""
}

func findMainContent(doc *html.Node) string {
	type candidate struct {
		node  *html.Node
		score float64
	}

	var candidates []candidate

	if article := findFirstNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "article"
	}); article != nil {
		candidates = append(candidates, candidate{article, calculateScore(article)})
	}

	if main := findFirstNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "main"
	}); main != nil {
		candidates = append(candidates, candidate{main, calculateScore(main)})
	}

	walkNodes(doc, func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			if getAttr(n, "role") == "main" {
				candidates = append(candidates, candidate{n, calculateScore(n)})
			} else {
				score := divCandidateScore(n)
				if score != nil {
					candidates = append(candidates, candidate{n, *score})
				}
			}
		}
	})

	if len(candidates) == 0 {
		if body := findFirstNode(doc, func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == "body"
		}); body != nil {
			return getTextContent(body)
		}
		return ""
	}

	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.score > best.score {
			best = c
		}
	}

	return getTextContent(best.node)
}

func divCandidateScore(div *html.Node) *float64 {
	divID := getAttr(div, "id")
	divClass := getAttr(div, "class")

	if divID == "" && divClass == "" {
		return nil
	}

	classID := strings.ToLower(divID + " " + divClass)
	boost := false
	penalty := false

	for _, cls := range boostClasses {
		if strings.Contains(classID, cls) {
			boost = true
			break
		}
	}

	for _, cls := range penaltyClasses {
		if strings.Contains(classID, cls) {
			penalty = true
			break
		}
	}

	if penalty && !boost {
		return nil
	}

	score := calculateScore(div)
	if boost {
		score += 2.0
	}
	if penalty {
		score -= 2.0
	}

	return &score
}

func calculateScore(node *html.Node) float64 {
	text := getTextContent(node)
	if text == "" {
		return 0.0
	}

	textLength := len(text)
	linkTextLength := sumLinkText(node)

	if textLength == 0 {
		return 0.0
	}

	linkDensity := float64(linkTextLength) / float64(textLength)
	textDensity := 1.0 - linkDensity

	score := textDensity * 10.0
	score += math.Log(float64(textLength)) / 2.0

	return score
}

func sumLinkText(node *html.Node) int {
	total := 0
	walkNodes(node, func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			total += len(getTextContent(n))
		}
	})
	return total
}

func cleanContent(text string) string {
	spaceRe := regexp.MustCompile(`\s+`)
	text = spaceRe.ReplaceAllString(text, " ")

	newlineRe := regexp.MustCompile(`\n\s*\n`)
	text = newlineRe.ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func getTextContent(n *html.Node) string {
	var buf bytes.Buffer
	var walker func(*html.Node)
	walker = func(node *html.Node) {
		if node.Type == html.TextNode {
			buf.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walker(c)
		}
	}
	walker(n)
	return buf.String()
}

func findFirstNode(root *html.Node, predicate func(*html.Node) bool) *html.Node {
	var result *html.Node
	var walker func(*html.Node)
	walker = func(n *html.Node) {
		if result != nil {
			return
		}
		if predicate(n) {
			result = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walker(c)
			if result != nil {
				return
			}
		}
	}
	walker(root)
	return result
}

func walkNodes(n *html.Node, fn func(*html.Node)) {
	fn(n)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkNodes(c, fn)
	}
}
