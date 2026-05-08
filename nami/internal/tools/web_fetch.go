package tools

import (
	"bytes"
	"container/list"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	xhtml "golang.org/x/net/html"
)

const (
	defaultWebFetchTimeout         = 60 * time.Second
	maxWebFetchURLLength           = 2000
	maxWebFetchContentBytes  int64 = 10 * 1024 * 1024
	maxWebFetchMarkdownChars       = 100_000
	maxWebFetchCacheBytes    int64 = 50 * 1024 * 1024
	webFetchCacheTTL               = 15 * time.Minute
	maxWebFetchRedirects           = 10
	webFetchUserAgent              = "nami/0.1 (+https://github.com/channyeintun/nami)"
)

type webFetchRespondMode string

const (
	webFetchRespondModeReport   webFetchRespondMode = "report"
	webFetchRespondModeMarkdown webFetchRespondMode = "markdown"
)

var webFetchContentHints = []string{"article", "content", "entry", "main", "post", "story"}
var webFetchNegativeHints = []string{"advert", "banner", "breadcrumb", "comment", "consent", "cookie", "footer", "menu", "modal", "nav", "promo", "related", "share", "sidebar", "social", "subscribe"}
var webFetchReadabilityUnlikelyPattern = regexp.MustCompile(`-ad-|banner|breadcrumbs|combx|comment|community|disqus|extra|footer|gdpr|header|menu|pagination|pager|popup|related|remark|replies|rss|shoutbox|sidebar|skyscraper|social|sponsor|supplemental`)
var webFetchReadabilityMaybePattern = regexp.MustCompile(`and|article|body|column|content|main|shadow`)
var webFetchReadabilityPositivePattern = regexp.MustCompile(`article|body|content|entry|hentry|h-entry|main|page|post|text|blog|story`)
var webFetchReadabilityNegativePattern = regexp.MustCompile(`-ad-|hidden|banner|combx|comment|contact|footer|gdpr|masthead|media|meta|promo|related|scroll|share|shoutbox|sidebar|skyscraper|sponsor|shopping|tags|widget`)
var webFetchReadabilityCommaPattern = regexp.MustCompile(`[,0CE50E10E11E41E34E32F0C]`)
var webFetchReadabilitySentencePattern = regexp.MustCompile(`\.( |$)`)
var webFetchReadabilityAdWordPattern = regexp.MustCompile(`^(ad|advertising|advertisement|promo)$`)
var webFetchReadabilityLoadingPattern = regexp.MustCompile(`^(loading|loading\.\.\.|loading…)$`)
var webFetchAlwaysDropTags = map[string]struct{}{
	"button":   {},
	"canvas":   {},
	"dialog":   {},
	"form":     {},
	"iframe":   {},
	"input":    {},
	"link":     {},
	"meta":     {},
	"noscript": {},
	"option":   {},
	"script":   {},
	"select":   {},
	"style":    {},
	"svg":      {},
	"template": {},
	"textarea": {},
}
var webFetchChromeTags = map[string]struct{}{
	"aside":  {},
	"footer": {},
	"nav":    {},
}
var webFetchReadabilityScorableTags = map[string]struct{}{
	"h2":      {},
	"h3":      {},
	"h4":      {},
	"h5":      {},
	"h6":      {},
	"p":       {},
	"pre":     {},
	"section": {},
	"td":      {},
}
var webFetchReadabilityContainerTags = map[string]struct{}{
	"div":     {},
	"header":  {},
	"h1":      {},
	"h2":      {},
	"h3":      {},
	"h4":      {},
	"h5":      {},
	"h6":      {},
	"section": {},
}
var webFetchReadabilityBlockTags = map[string]struct{}{
	"article":    {},
	"aside":      {},
	"blockquote": {},
	"div":        {},
	"dl":         {},
	"figure":     {},
	"footer":     {},
	"header":     {},
	"img":        {},
	"main":       {},
	"nav":        {},
	"ol":         {},
	"p":          {},
	"pre":        {},
	"section":    {},
	"table":      {},
	"ul":         {},
}
var webFetchReadabilityMediaTags = map[string]struct{}{
	"audio":  {},
	"embed":  {},
	"iframe": {},
	"img":    {},
	"object": {},
	"video":  {},
}
var webFetchReadabilityUnlikelyRoles = map[string]struct{}{
	"alert":         {},
	"alertdialog":   {},
	"complementary": {},
	"dialog":        {},
	"menu":          {},
	"menubar":       {},
	"navigation":    {},
}
var webFetchReadabilityHeadingTags = map[string]struct{}{
	"h1": {},
	"h2": {},
	"h3": {},
	"h4": {},
	"h5": {},
	"h6": {},
}
var webFetchReadabilityTextishTags = map[string]struct{}{
	"blockquote": {},
	"div":        {},
	"dl":         {},
	"img":        {},
	"li":         {},
	"ol":         {},
	"p":          {},
	"pre":        {},
	"span":       {},
	"table":      {},
	"td":         {},
	"ul":         {},
}

// WebFetchTool fetches a URL, converts HTML to markdown, and returns prompt-focused content.
type WebFetchTool struct {
	client *http.Client
	cache  *webFetchCache
}

// NewWebFetchTool constructs the web fetch tool.
func NewWebFetchTool() *WebFetchTool {
	return &WebFetchTool{
		client: &http.Client{
			Timeout: defaultWebFetchTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= maxWebFetchRedirects {
					return errors.New("stopped after too many redirects")
				}
				return http.ErrUseLastResponse
			},
		},
		cache: newWebFetchCache(maxWebFetchCacheBytes, webFetchCacheTTL),
	}
}

func (t *WebFetchTool) Name() string {
	return "web_fetch"
}

func (t *WebFetchTool) Description() string {
	return "Fetch content from a URL, convert HTML to markdown, and return either prompt-focused excerpts or markdown."
}

func (t *WebFetchTool) InputSchema() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The URL to fetch.",
			},
			"prompt": map[string]any{
				"type":        "string",
				"description": "What information to extract from the fetched content. Required unless respond_with is markdown.",
			},
			"respond_with": map[string]any{
				"type":        "string",
				"description": "Optional output mode. Use markdown to return the converted page markdown directly.",
				"enum":        []string{"report", "markdown"},
			},
			"respondWith": map[string]any{
				"type":        "string",
				"description": "CamelCase alias for respond_with.",
				"enum":        []string{"report", "markdown"},
			},
		},
		"required": []string{"url"},
	}
}

func (t *WebFetchTool) Permission() PermissionLevel {
	return PermissionReadOnly
}

func (t *WebFetchTool) Concurrency(input ToolInput) ConcurrencyDecision {
	return ConcurrencyParallel
}

func (t *WebFetchTool) Execute(ctx context.Context, input ToolInput) (ToolOutput, error) {
	rawURL, ok := stringParam(input.Params, "url")
	if !ok || strings.TrimSpace(rawURL) == "" {
		return ToolOutput{}, fmt.Errorf("web_fetch requires url")
	}
	respondMode, err := parseWebFetchRespondMode(input.Params)
	if err != nil {
		return ToolOutput{}, err
	}
	prompt, err := parseWebFetchPrompt(input.Params, respondMode)
	if err != nil {
		return ToolOutput{}, err
	}

	normalizedURL, err := normalizeWebFetchURL(rawURL)
	if err != nil {
		return ToolOutput{}, err
	}

	content, err := t.getMarkdownContent(ctx, normalizedURL)
	if err != nil {
		return ToolOutput{}, err
	}

	result := buildWebFetchResult(normalizedURL, prompt, respondMode, content)

	// Route substantial fetch results to a search-report artifact.
	const searchReportThreshold = 4000
	if respondMode == webFetchRespondModeReport && len(result) >= searchReportThreshold {
		if mutation, ok := saveSearchReportArtifact(ctx, normalizedURL, strings.TrimSpace(prompt), result); ok {
			return ToolOutput{Output: result, Artifacts: []ArtifactMutation{mutation}}, nil
		}
	}

	return ToolOutput{Output: result}, nil
}

type webFetchContent struct {
	URL         string
	StatusCode  int
	StatusText  string
	ContentType string
	Bytes       int
	Markdown    string
}

func (t *WebFetchTool) getMarkdownContent(ctx context.Context, rawURL string) (webFetchContent, error) {
	if cached, ok := t.cache.Get(rawURL); ok {
		return cached, nil
	}

	currentURL := rawURL
	for redirectCount := 0; redirectCount <= maxWebFetchRedirects; redirectCount++ {
		content, redirectURL, err := t.fetchOnce(ctx, currentURL)
		if err != nil {
			return webFetchContent{}, err
		}
		if redirectURL == "" {
			t.cache.Set(rawURL, content)
			return content, nil
		}
		if !webFetchPermittedRedirect(currentURL, redirectURL) {
			return webFetchContent{}, fmt.Errorf("web_fetch redirect requires approval: %s -> %s", currentURL, redirectURL)
		}
		currentURL = redirectURL
	}

	return webFetchContent{}, fmt.Errorf("web_fetch exceeded %d redirects", maxWebFetchRedirects)
}

func (t *WebFetchTool) fetchOnce(ctx context.Context, rawURL string) (webFetchContent, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return webFetchContent{}, "", fmt.Errorf("create web fetch request: %w", err)
	}
	req.Header.Set("Accept", "text/markdown, text/html, text/plain, */*")
	req.Header.Set("User-Agent", webFetchUserAgent)

	resp, err := t.client.Do(req)
	if err != nil {
		return webFetchContent{}, "", fmt.Errorf("execute web fetch request: %w", err)
	}
	defer resp.Body.Close()

	if isRedirectStatus(resp.StatusCode) {
		location := strings.TrimSpace(resp.Header.Get("Location"))
		if location == "" {
			return webFetchContent{}, "", fmt.Errorf("redirect missing Location header")
		}
		redirectURL, err := resolveWebFetchRedirect(rawURL, location)
		if err != nil {
			return webFetchContent{}, "", err
		}
		return webFetchContent{}, redirectURL, nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return webFetchContent{}, "", fmt.Errorf("web_fetch returned status %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxWebFetchContentBytes+1))
	if err != nil {
		return webFetchContent{}, "", fmt.Errorf("read web fetch response: %w", err)
	}
	if int64(len(body)) > maxWebFetchContentBytes {
		return webFetchContent{}, "", fmt.Errorf("web_fetch response exceeded %d bytes", maxWebFetchContentBytes)
	}

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	markdown, err := t.toMarkdown(string(body), contentType)
	if err != nil {
		return webFetchContent{}, "", err
	}

	return webFetchContent{
		URL:         rawURL,
		StatusCode:  resp.StatusCode,
		StatusText:  resp.Status,
		ContentType: contentType,
		Bytes:       len(body),
		Markdown:    markdown,
	}, "", nil
}

func (t *WebFetchTool) toMarkdown(body string, contentType string) (string, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", nil
	}

	if strings.Contains(contentType, "text/html") || looksLikeHTML(body) {
		focusedHTML := extractWebFetchHTMLForMarkdown(body)
		markdown, err := htmltomarkdown.ConvertString(focusedHTML)
		if err != nil {
			return "", fmt.Errorf("convert html to markdown: %w", err)
		}
		return strings.TrimSpace(markdown), nil
	}

	return body, nil
}

func buildWebFetchResult(rawURL, prompt string, respondMode webFetchRespondMode, content webFetchContent) string {
	truncatedContent := truncateWebFetchMarkdown(content.Markdown)
	if respondMode == webFetchRespondModeMarkdown {
		if truncatedContent == "" {
			return "No readable content returned."
		}
		return truncatedContent
	}

	passages := extractRelevantPassages(truncatedContent, prompt)

	var builder strings.Builder
	fmt.Fprintf(&builder, "Fetched: %s\n", rawURL)
	fmt.Fprintf(&builder, "Status: %s\n", content.StatusText)
	if content.ContentType != "" {
		fmt.Fprintf(&builder, "Content-Type: %s\n", content.ContentType)
	}
	fmt.Fprintf(&builder, "Bytes: %d\n", content.Bytes)
	fmt.Fprintf(&builder, "Prompt: %s\n", strings.TrimSpace(prompt))

	if len(passages) > 0 {
		builder.WriteString("\nRelevant excerpts:\n")
		for index, passage := range passages {
			fmt.Fprintf(&builder, "\n%d. %s\n", index+1, passage)
		}
	} else if truncatedContent != "" {
		builder.WriteString("\nContent:\n\n")
		builder.WriteString(truncatedContent)
	} else {
		builder.WriteString("\nNo readable content returned.\n")
	}

	return strings.TrimSpace(builder.String())
}

func truncateWebFetchMarkdown(markdown string) string {
	markdown = strings.TrimSpace(markdown)
	if len(markdown) <= maxWebFetchMarkdownChars {
		return markdown
	}
	return markdown[:maxWebFetchMarkdownChars] + "\n\n[Content truncated due to length...]"
}

func parseWebFetchRespondMode(params map[string]any) (webFetchRespondMode, error) {
	value, ok := firstStringParam(params, "respond_with", "respondWith")
	if !ok {
		return webFetchRespondModeReport, nil
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", string(webFetchRespondModeReport):
		return webFetchRespondModeReport, nil
	case string(webFetchRespondModeMarkdown):
		return webFetchRespondModeMarkdown, nil
	default:
		return "", fmt.Errorf("web_fetch respond_with must be report or markdown")
	}
}

func parseWebFetchPrompt(params map[string]any, respondMode webFetchRespondMode) (string, error) {
	prompt, _ := stringParam(params, "prompt")
	prompt = strings.TrimSpace(prompt)
	if respondMode == webFetchRespondModeMarkdown {
		return prompt, nil
	}
	if prompt == "" {
		return "", fmt.Errorf("web_fetch requires prompt")
	}
	return prompt, nil
}

func extractWebFetchHTMLForMarkdown(body string) string {
	document, err := xhtml.Parse(strings.NewReader(body))
	if err != nil {
		return body
	}

	root := findFirstWebFetchElement(document, "body")
	if root == nil {
		root = document
	}
	selected := selectWebFetchContentRoot(root)
	pruneWebFetchChrome(selected)
	prepareWebFetchArticle(selected)

	var builder bytes.Buffer
	if err := xhtml.Render(&builder, selected); err != nil {
		return body
	}
	return builder.String()
}

func findFirstWebFetchElement(root *xhtml.Node, tagName string) *xhtml.Node {
	if root == nil {
		return nil
	}
	if root.Type == xhtml.ElementNode && strings.EqualFold(root.Data, tagName) {
		return root
	}
	for child := root.FirstChild; child != nil; child = child.NextSibling {
		if found := findFirstWebFetchElement(child, tagName); found != nil {
			return found
		}
	}
	return nil
}

func selectWebFetchContentRoot(root *xhtml.Node) *xhtml.Node {
	if candidate := extractWebFetchReadableContent(root); candidate != nil {
		return candidate
	}
	return selectWebFetchHeuristicContentRoot(root)
}

func selectWebFetchHeuristicContentRoot(root *xhtml.Node) *xhtml.Node {
	if root == nil {
		return nil
	}
	bodyText, _ := webFetchTextMetrics(root, false)
	if bodyText == 0 {
		return root
	}

	bestNode := root
	bestScore := bodyText
	for _, candidate := range collectWebFetchContentCandidates(root) {
		visibleText, linkText := webFetchTextMetrics(candidate, false)
		if visibleText < 200 {
			continue
		}
		if visibleText*100 < bodyText*35 {
			continue
		}
		score := visibleText - linkText
		if score > bestScore {
			bestNode = candidate
			bestScore = score
		}
	}
	return bestNode
}

func extractWebFetchReadableContent(root *xhtml.Node) *xhtml.Node {
	if root == nil {
		return nil
	}

	elementsToScore := collectWebFetchReadabilityNodes(root)
	if len(elementsToScore) == 0 {
		return nil
	}

	baseScores := make(map[*xhtml.Node]float64)
	adjustedScores := make(map[*xhtml.Node]float64)
	candidates := make([]*xhtml.Node, 0, len(elementsToScore))

	for _, node := range elementsToScore {
		innerText := webFetchInnerText(node, true)
		if len(innerText) < 25 {
			continue
		}

		ancestors := webFetchNodeAncestors(node, 5)
		if len(ancestors) == 0 {
			continue
		}

		contentScore := 1.0
		contentScore += float64(len(webFetchReadabilityCommaPattern.FindAllStringIndex(innerText, -1)) + 1)
		contentScore += float64(min(len(innerText)/100, 3))

		for depth, ancestor := range ancestors {
			if ancestor == nil || ancestor.Type != xhtml.ElementNode {
				continue
			}
			if _, ok := baseScores[ancestor]; !ok {
				baseScores[ancestor] = webFetchInitializeReadabilityScore(ancestor)
				candidates = append(candidates, ancestor)
			}

			divider := 1.0
			switch depth {
			case 0:
				divider = 1
			case 1:
				divider = 2
			default:
				divider = float64(depth * 3)
			}
			baseScores[ancestor] += contentScore / divider
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	topCandidates := make([]*xhtml.Node, 0, 5)
	for _, candidate := range candidates {
		score := baseScores[candidate] * (1 - webFetchLinkDensity(candidate))
		adjustedScores[candidate] = score
		topCandidates = insertWebFetchTopCandidate(topCandidates, candidate, adjustedScores, 5)
	}

	if len(topCandidates) == 0 {
		return nil
	}

	topCandidate := refineWebFetchTopCandidate(topCandidates[0], adjustedScores)
	if topCandidate == nil || adjustedScores[topCandidate] <= 0 {
		return nil
	}

	article := collectWebFetchReadableSiblings(topCandidate, adjustedScores)
	if article == nil {
		return topCandidate
	}
	if len(webFetchInnerText(article, true)) < 140 {
		return topCandidate
	}
	return article
}

func collectWebFetchReadabilityNodes(root *xhtml.Node) []*xhtml.Node {
	if root == nil {
		return nil
	}

	var nodes []*xhtml.Node
	var walk func(node *xhtml.Node)
	walk = func(node *xhtml.Node) {
		if node == nil {
			return
		}
		if node.Type == xhtml.ElementNode {
			if !webFetchIsProbablyVisible(node) || webFetchShouldSkipReadabilityNode(node) {
				return
			}
			tagName := webFetchElementName(node)
			if _, ok := webFetchReadabilityContainerTags[tagName]; ok && webFetchIsElementWithoutContent(node) {
				return
			}
			if _, ok := webFetchReadabilityScorableTags[tagName]; ok {
				nodes = append(nodes, node)
			}
			if tagName == "div" && !webFetchHasChildBlockElement(node) {
				nodes = append(nodes, node)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	return nodes
}

func insertWebFetchTopCandidate(topCandidates []*xhtml.Node, candidate *xhtml.Node, scores map[*xhtml.Node]float64, limit int) []*xhtml.Node {
	inserted := false
	for index, existing := range topCandidates {
		if scores[candidate] > scores[existing] {
			topCandidates = append(topCandidates, nil)
			copy(topCandidates[index+1:], topCandidates[index:])
			topCandidates[index] = candidate
			inserted = true
			break
		}
	}
	if !inserted && len(topCandidates) < limit {
		topCandidates = append(topCandidates, candidate)
	}
	if len(topCandidates) > limit {
		topCandidates = topCandidates[:limit]
	}
	return topCandidates
}

func refineWebFetchTopCandidate(topCandidate *xhtml.Node, scores map[*xhtml.Node]float64) *xhtml.Node {
	if topCandidate == nil {
		return nil
	}

	lastScore := scores[topCandidate]
	scoreFloor := lastScore / 3
	for parent := topCandidate.Parent; parent != nil && webFetchElementName(parent) != "body"; parent = parent.Parent {
		parentScore, ok := scores[parent]
		if !ok {
			continue
		}
		if parentScore < scoreFloor {
			break
		}
		if parentScore > lastScore {
			topCandidate = parent
			break
		}
		lastScore = parentScore
	}

	for parent := topCandidate.Parent; parent != nil && webFetchElementName(parent) != "body" && webFetchElementChildCount(parent) == 1; parent = topCandidate.Parent {
		topCandidate = parent
	}
	return topCandidate
}

func collectWebFetchReadableSiblings(topCandidate *xhtml.Node, scores map[*xhtml.Node]float64) *xhtml.Node {
	if topCandidate == nil || topCandidate.Parent == nil {
		return nil
	}

	parent := topCandidate.Parent
	article := &xhtml.Node{Type: xhtml.ElementNode, Data: "div"}
	threshold := scores[topCandidate] * 0.2
	if threshold < 10 {
		threshold = 10
	}
	topClass := strings.TrimSpace(webFetchAttr(topCandidate, "class"))

	for sibling := parent.FirstChild; sibling != nil; {
		nextSibling := sibling.NextSibling
		shouldAppend := sibling == topCandidate

		if !shouldAppend && sibling.Type == xhtml.ElementNode {
			bonus := 0.0
			if topClass != "" && strings.TrimSpace(webFetchAttr(sibling, "class")) == topClass {
				bonus = scores[topCandidate] * 0.2
			}
			if scores[sibling]+bonus >= threshold {
				shouldAppend = true
			} else if webFetchElementName(sibling) == "p" {
				nodeContent := webFetchInnerText(sibling, true)
				nodeLength := len(nodeContent)
				linkDensity := webFetchLinkDensity(sibling)
				if nodeLength > 80 && linkDensity < 0.25 {
					shouldAppend = true
				} else if nodeLength > 0 && nodeLength < 80 && linkDensity == 0 && webFetchReadabilitySentencePattern.MatchString(nodeContent) {
					shouldAppend = true
				}
			}
		}

		if shouldAppend {
			parent.RemoveChild(sibling)
			article.AppendChild(sibling)
		}
		sibling = nextSibling
	}

	if article.FirstChild == nil {
		return nil
	}
	return article
}

func webFetchInitializeReadabilityScore(node *xhtml.Node) float64 {
	score := 0.0
	switch webFetchElementName(node) {
	case "div":
		score += 5
	case "blockquote", "pre", "td":
		score += 3
	case "address", "dd", "dl", "dt", "form", "li", "ol", "ul":
		score -= 3
	case "h1", "h2", "h3", "h4", "h5", "h6", "th":
		score -= 5
	}
	return score + webFetchClassWeight(node)
}

func webFetchClassWeight(node *xhtml.Node) float64 {
	marker := strings.TrimSpace(webFetchAttr(node, "class"))
	id := strings.TrimSpace(webFetchAttr(node, "id"))
	weight := 0.0
	if marker != "" {
		if webFetchReadabilityNegativePattern.MatchString(marker) {
			weight -= 25
		}
		if webFetchReadabilityPositivePattern.MatchString(marker) {
			weight += 25
		}
	}
	if id != "" {
		if webFetchReadabilityNegativePattern.MatchString(id) {
			weight -= 25
		}
		if webFetchReadabilityPositivePattern.MatchString(id) {
			weight += 25
		}
	}
	return weight
}

func webFetchLinkDensity(node *xhtml.Node) float64 {
	textLength := len(webFetchInnerText(node, true))
	if textLength == 0 {
		return 0
	}

	linkText := 0.0
	var walk func(current *xhtml.Node)
	walk = func(current *xhtml.Node) {
		if current == nil {
			return
		}
		if current.Type == xhtml.ElementNode && webFetchElementName(current) == "a" {
			coefficient := 1.0
			href := strings.TrimSpace(webFetchAttr(current, "href"))
			if strings.HasPrefix(href, "#") {
				coefficient = 0.3
			}
			linkText += float64(len(webFetchInnerText(current, true))) * coefficient
			return
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return linkText / float64(textLength)
}

func webFetchNodeAncestors(node *xhtml.Node, maxDepth int) []*xhtml.Node {
	ancestors := make([]*xhtml.Node, 0, maxDepth)
	for parent := node.Parent; parent != nil; parent = parent.Parent {
		if parent.Type == xhtml.ElementNode {
			ancestors = append(ancestors, parent)
			if maxDepth > 0 && len(ancestors) == maxDepth {
				break
			}
		}
	}
	return ancestors
}

func webFetchInnerText(node *xhtml.Node, normalizeSpaces bool) string {
	if node == nil {
		return ""
	}

	var builder strings.Builder
	var walk func(current *xhtml.Node)
	walk = func(current *xhtml.Node) {
		if current == nil {
			return
		}
		switch current.Type {
		case xhtml.TextNode:
			builder.WriteString(current.Data)
			builder.WriteByte(' ')
		case xhtml.ElementNode:
			if _, ok := webFetchAlwaysDropTags[webFetchElementName(current)]; ok {
				return
			}
			for child := current.FirstChild; child != nil; child = child.NextSibling {
				walk(child)
			}
			if _, ok := webFetchReadabilityBlockTags[webFetchElementName(current)]; ok {
				builder.WriteByte(' ')
			}
		}
	}
	walk(node)
	text := strings.TrimSpace(builder.String())
	if !normalizeSpaces {
		return text
	}
	return strings.Join(strings.Fields(text), " ")
}

func webFetchIsProbablyVisible(node *xhtml.Node) bool {
	if node == nil {
		return false
	}
	if _, hidden := webFetchNodeAttr(node, "hidden"); hidden {
		return false
	}
	if ariaHidden, ok := webFetchNodeAttr(node, "aria-hidden"); ok && strings.EqualFold(strings.TrimSpace(ariaHidden), "true") && !strings.Contains(strings.ToLower(webFetchAttr(node, "class")), "fallback-image") {
		return false
	}
	if style, ok := webFetchNodeAttr(node, "style"); ok {
		style = strings.ToLower(style)
		if strings.Contains(style, "display:none") || strings.Contains(style, "display: none") || strings.Contains(style, "visibility:hidden") || strings.Contains(style, "visibility: hidden") {
			return false
		}
	}
	return true
}

func webFetchShouldSkipReadabilityNode(node *xhtml.Node) bool {
	if node == nil || node.Type != xhtml.ElementNode {
		return false
	}
	role := strings.ToLower(strings.TrimSpace(webFetchAttr(node, "role")))
	if _, ok := webFetchReadabilityUnlikelyRoles[role]; ok {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(webFetchAttr(node, "aria-modal")), "true") && role == "dialog" {
		return true
	}

	matchString := strings.ToLower(strings.TrimSpace(webFetchAttr(node, "class") + " " + webFetchAttr(node, "id")))
	tagName := webFetchElementName(node)
	if webFetchReadabilityUnlikelyPattern.MatchString(matchString) &&
		!webFetchReadabilityMaybePattern.MatchString(matchString) &&
		!webFetchHasAncestorTag(node, "table") &&
		!webFetchHasAncestorTag(node, "code") &&
		tagName != "body" &&
		tagName != "a" {
		return true
	}
	return false
}

func webFetchHasAncestorTag(node *xhtml.Node, tagName string) bool {
	for parent := node.Parent; parent != nil; parent = parent.Parent {
		if webFetchElementName(parent) == tagName {
			return true
		}
	}
	return false
}

func webFetchIsElementWithoutContent(node *xhtml.Node) bool {
	if node == nil {
		return true
	}
	if strings.TrimSpace(webFetchInnerText(node, true)) != "" {
		return false
	}
	return !webFetchHasDescendantTag(node, webFetchReadabilityMediaTags)
}

func webFetchHasDescendantTag(node *xhtml.Node, tagSet map[string]struct{}) bool {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == xhtml.ElementNode {
			if _, ok := tagSet[webFetchElementName(child)]; ok {
				return true
			}
		}
		if webFetchHasDescendantTag(child, tagSet) {
			return true
		}
	}
	return false
}

func webFetchHasChildBlockElement(node *xhtml.Node) bool {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != xhtml.ElementNode {
			continue
		}
		if _, ok := webFetchReadabilityBlockTags[webFetchElementName(child)]; ok {
			return true
		}
	}
	return false
}

func webFetchElementChildCount(node *xhtml.Node) int {
	count := 0
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == xhtml.ElementNode {
			count++
		}
	}
	return count
}

func webFetchElementName(node *xhtml.Node) string {
	if node == nil || node.Type != xhtml.ElementNode {
		return ""
	}
	return strings.ToLower(node.Data)
}

func webFetchNodeAttr(node *xhtml.Node, key string) (string, bool) {
	if node == nil {
		return "", false
	}
	for _, attribute := range node.Attr {
		if strings.EqualFold(attribute.Key, key) {
			return attribute.Val, true
		}
	}
	return "", false
}

func collectWebFetchContentCandidates(root *xhtml.Node) []*xhtml.Node {
	if root == nil {
		return nil
	}
	var candidates []*xhtml.Node
	var visit func(node *xhtml.Node)
	visit = func(node *xhtml.Node) {
		if node.Type == xhtml.ElementNode && isWebFetchContentCandidate(node) {
			candidates = append(candidates, node)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(root)
	return candidates
}

func isWebFetchContentCandidate(node *xhtml.Node) bool {
	if node == nil || node.Type != xhtml.ElementNode {
		return false
	}
	if node.Data == "main" || node.Data == "article" {
		return true
	}
	role := strings.ToLower(strings.TrimSpace(webFetchAttr(node, "role")))
	if role == "main" || role == "article" {
		return true
	}
	marker := strings.ToLower(webFetchAttr(node, "id") + " " + webFetchAttr(node, "class"))
	if marker == "" || containsAnyWebFetchHint(marker, webFetchNegativeHints) {
		return false
	}
	return containsAnyWebFetchHint(marker, webFetchContentHints)
}

func pruneWebFetchChrome(root *xhtml.Node) {
	if root == nil {
		return
	}
	for child := root.FirstChild; child != nil; {
		nextChild := child.NextSibling
		if shouldPruneWebFetchNode(child) {
			root.RemoveChild(child)
			child = nextChild
			continue
		}
		pruneWebFetchChrome(child)
		child = nextChild
	}
}

func shouldPruneWebFetchNode(node *xhtml.Node) bool {
	if node == nil || node.Type != xhtml.ElementNode {
		return false
	}
	tagName := strings.ToLower(node.Data)
	if _, ok := webFetchAlwaysDropTags[tagName]; ok {
		return true
	}
	if _, ok := webFetchChromeTags[tagName]; ok {
		return true
	}
	if nodeHasHiddenWebFetchAttrs(node) {
		return true
	}
	marker := strings.ToLower(webFetchAttr(node, "id") + " " + webFetchAttr(node, "class") + " " + webFetchAttr(node, "role") + " " + webFetchAttr(node, "aria-label"))
	return containsAnyWebFetchHint(marker, webFetchNegativeHints)
}

func prepareWebFetchArticle(root *xhtml.Node) {
	if root == nil {
		return
	}
	for _, tagName := range []string{"form", "fieldset"} {
		webFetchCleanConditionally(root, tagName)
	}
	for _, tagName := range []string{"object", "embed", "footer", "link", "aside", "iframe", "input", "textarea", "select", "button"} {
		webFetchRemoveTag(root, tagName)
	}
	webFetchCleanHeaders(root)
	for _, tagName := range []string{"table", "ul", "div"} {
		webFetchCleanConditionally(root, tagName)
	}
	webFetchRenameTag(root, "h1", "h2")
	webFetchRemoveEmptyParagraphs(root)
	webFetchCollapseSingleCellTables(root)
}

func webFetchRemoveTag(root *xhtml.Node, tagName string) {
	for _, node := range webFetchCollectNodesByTag(root, tagName) {
		if node.Parent != nil {
			node.Parent.RemoveChild(node)
		}
	}
}

func webFetchCleanHeaders(root *xhtml.Node) {
	for _, node := range webFetchCollectNodesByTag(root, "h1", "h2") {
		if node.Parent != nil && webFetchClassWeight(node) < 0 {
			node.Parent.RemoveChild(node)
		}
	}
}

func webFetchCleanConditionally(root *xhtml.Node, tagName string) {
	for _, node := range webFetchCollectNodesByTag(root, tagName) {
		if node == nil || node.Parent == nil {
			continue
		}
		if tagName == "table" && webFetchIsDataTable(node) {
			continue
		}
		if webFetchHasAncestorMatching(node, "table", webFetchIsDataTable) || webFetchHasAncestorTag(node, "code") || webFetchSubtreeHasDataTable(node) {
			continue
		}

		weight := webFetchClassWeight(node)
		if weight < 0 {
			node.Parent.RemoveChild(node)
			continue
		}
		if webFetchCharCount(node, ",") >= 10 {
			continue
		}

		innerText := webFetchInnerText(node, true)
		if innerText == "" {
			node.Parent.RemoveChild(node)
			continue
		}
		normalizedText := strings.ToLower(strings.TrimSpace(innerText))
		if webFetchReadabilityAdWordPattern.MatchString(normalizedText) || webFetchReadabilityLoadingPattern.MatchString(normalizedText) {
			node.Parent.RemoveChild(node)
			continue
		}

		paragraphCount := len(webFetchCollectNodesByTag(node, "p"))
		imageCount := len(webFetchCollectNodesByTag(node, "img"))
		listItemCount := len(webFetchCollectNodesByTag(node, "li")) - 100
		inputCount := len(webFetchCollectNodesByTag(node, "input"))
		embedCount := len(webFetchCollectNodesByTag(node, "object", "embed", "iframe"))
		contentLength := len(innerText)
		linkDensity := webFetchLinkDensity(node)
		headingDensity := webFetchTextDensity(node, webFetchReadabilityHeadingTags)
		textDensity := webFetchTextDensity(node, webFetchReadabilityTextishTags)
		isList := tagName == "ul" || tagName == "ol" || webFetchListDensity(node) > 0.9
		isFigureChild := webFetchHasAncestorTag(node, "figure")

		shouldRemove := false
		if !isFigureChild && imageCount > 1 && float64(paragraphCount) < float64(imageCount)/2 {
			shouldRemove = true
		}
		if !isList && listItemCount > paragraphCount {
			shouldRemove = true
		}
		if inputCount > paragraphCount/3 {
			shouldRemove = true
		}
		if !isList && !isFigureChild && headingDensity < 0.9 && contentLength < 25 && (imageCount == 0 || imageCount > 2) && linkDensity > 0 {
			shouldRemove = true
		}
		if !isList && weight < 25 && linkDensity > 0.2 {
			shouldRemove = true
		}
		if weight >= 25 && linkDensity > 0.5 {
			shouldRemove = true
		}
		if (embedCount == 1 && contentLength < 75) || embedCount > 1 {
			shouldRemove = true
		}
		if imageCount == 0 && textDensity == 0 {
			shouldRemove = true
		}

		if shouldRemove && !(isList && webFetchLooksLikeImageList(node, imageCount)) {
			node.Parent.RemoveChild(node)
		}
	}
}

func webFetchTextDensity(node *xhtml.Node, tagSet map[string]struct{}) float64 {
	textLength := len(webFetchInnerText(node, true))
	if textLength == 0 {
		return 0
	}
	childTextLength := 0
	for _, child := range webFetchCollectNodes(node, func(candidate *xhtml.Node) bool {
		_, ok := tagSet[webFetchElementName(candidate)]
		return ok
	}) {
		childTextLength += len(webFetchInnerText(child, true))
	}
	return float64(childTextLength) / float64(textLength)
}

func webFetchListDensity(node *xhtml.Node) float64 {
	textLength := len(webFetchInnerText(node, true))
	if textLength == 0 {
		return 0
	}
	listLength := 0
	for _, list := range webFetchCollectNodesByTag(node, "ul", "ol") {
		listLength += len(webFetchInnerText(list, true))
	}
	return float64(listLength) / float64(textLength)
}

func webFetchLooksLikeImageList(node *xhtml.Node, imageCount int) bool {
	children := webFetchElementChildren(node)
	if len(children) == 0 {
		return false
	}
	for _, child := range children {
		if len(webFetchElementChildren(child)) > 1 {
			return false
		}
	}
	return imageCount == len(webFetchCollectNodesByTag(node, "li"))
}

func webFetchCharCount(node *xhtml.Node, target string) int {
	return strings.Count(webFetchInnerText(node, true), target)
}

func webFetchHasAncestorMatching(node *xhtml.Node, tagName string, predicate func(*xhtml.Node) bool) bool {
	for parent := node.Parent; parent != nil; parent = parent.Parent {
		if webFetchElementName(parent) == tagName && predicate(parent) {
			return true
		}
	}
	return false
}

func webFetchSubtreeHasDataTable(node *xhtml.Node) bool {
	for _, table := range webFetchCollectNodesByTag(node, "table") {
		if table != node && webFetchIsDataTable(table) {
			return true
		}
	}
	return false
}

func webFetchIsDataTable(node *xhtml.Node) bool {
	if webFetchElementName(node) != "table" {
		return false
	}
	if role, ok := webFetchNodeAttr(node, "role"); ok && strings.EqualFold(strings.TrimSpace(role), "presentation") {
		return false
	}
	if datatable, ok := webFetchNodeAttr(node, "datatable"); ok && strings.TrimSpace(datatable) == "0" {
		return false
	}
	if summary, ok := webFetchNodeAttr(node, "summary"); ok && strings.TrimSpace(summary) != "" {
		return true
	}
	for _, caption := range webFetchCollectNodesByTag(node, "caption") {
		if strings.TrimSpace(webFetchInnerText(caption, true)) != "" {
			return true
		}
	}
	for _, tagName := range []string{"col", "colgroup", "tfoot", "thead", "th"} {
		if len(webFetchCollectNodesByTag(node, tagName)) > 0 {
			return true
		}
	}
	for _, table := range webFetchCollectNodesByTag(node, "table") {
		if table != node {
			return false
		}
	}
	rows, columns := webFetchTableDimensions(node)
	if rows == 1 || columns == 1 {
		return false
	}
	if rows >= 10 || columns > 4 {
		return true
	}
	return rows*columns > 10
}

func webFetchTableDimensions(table *xhtml.Node) (int, int) {
	rows := 0
	maxColumns := 0
	for _, row := range webFetchCollectNodesByTag(table, "tr") {
		rows++
		columnsInRow := 0
		for _, cell := range webFetchElementChildren(row) {
			name := webFetchElementName(cell)
			if name != "td" && name != "th" {
				continue
			}
			colspan := 1
			if rawColspan, ok := webFetchNodeAttr(cell, "colspan"); ok {
				var parsed int
				_, err := fmt.Sscanf(strings.TrimSpace(rawColspan), "%d", &parsed)
				if err == nil && parsed > 1 {
					colspan = parsed
				}
			}
			columnsInRow += colspan
		}
		if columnsInRow > maxColumns {
			maxColumns = columnsInRow
		}
	}
	return rows, maxColumns
}

func webFetchRenameTag(root *xhtml.Node, fromTag string, toTag string) {
	for _, node := range webFetchCollectNodesByTag(root, fromTag) {
		node.Data = toTag
	}
}

func webFetchRemoveEmptyParagraphs(root *xhtml.Node) {
	for _, paragraph := range webFetchCollectNodesByTag(root, "p") {
		if paragraph.Parent == nil {
			continue
		}
		contentElementCount := len(webFetchCollectNodesByTag(paragraph, "img", "embed", "object", "iframe"))
		if contentElementCount == 0 && webFetchInnerText(paragraph, false) == "" {
			paragraph.Parent.RemoveChild(paragraph)
		}
	}
}

func webFetchCollapseSingleCellTables(root *xhtml.Node) {
	for _, table := range webFetchCollectNodesByTag(root, "table") {
		if table.Parent == nil || webFetchIsDataTable(table) {
			continue
		}
		body := table
		if webFetchHasSingleTagInsideElement(table, "tbody") {
			body = webFetchFirstElementChild(table)
		}
		if body == nil || !webFetchHasSingleTagInsideElement(body, "tr") {
			continue
		}
		row := webFetchFirstElementChild(body)
		if row == nil || !webFetchHasSingleTagInsideElement(row, "td") {
			continue
		}
		cell := webFetchFirstElementChild(row)
		replacementTag := "div"
		if !webFetchHasChildBlockElement(cell) {
			replacementTag = "p"
		}
		replacement := &xhtml.Node{Type: xhtml.ElementNode, Data: replacementTag}
		for cell.FirstChild != nil {
			replacement.AppendChild(cell.FirstChild)
		}
		table.Parent.InsertBefore(replacement, table)
		table.Parent.RemoveChild(table)
	}
}

func webFetchHasSingleTagInsideElement(node *xhtml.Node, tagName string) bool {
	children := webFetchElementChildren(node)
	if len(children) != 1 || webFetchElementName(children[0]) != tagName {
		return false
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == xhtml.TextNode && strings.TrimSpace(child.Data) != "" {
			return false
		}
	}
	return true
}

func webFetchFirstElementChild(node *xhtml.Node) *xhtml.Node {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == xhtml.ElementNode {
			return child
		}
	}
	return nil
}

func webFetchElementChildren(node *xhtml.Node) []*xhtml.Node {
	children := make([]*xhtml.Node, 0, webFetchElementChildCount(node))
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == xhtml.ElementNode {
			children = append(children, child)
		}
	}
	return children
}

func webFetchCollectNodesByTag(root *xhtml.Node, tagNames ...string) []*xhtml.Node {
	tagSet := make(map[string]struct{}, len(tagNames))
	for _, tagName := range tagNames {
		tagSet[strings.ToLower(tagName)] = struct{}{}
	}
	return webFetchCollectNodes(root, func(node *xhtml.Node) bool {
		_, ok := tagSet[webFetchElementName(node)]
		return ok
	})
}

func webFetchCollectNodes(root *xhtml.Node, predicate func(*xhtml.Node) bool) []*xhtml.Node {
	if root == nil {
		return nil
	}
	nodes := make([]*xhtml.Node, 0)
	var walk func(node *xhtml.Node)
	walk = func(node *xhtml.Node) {
		if node == nil {
			return
		}
		if node.Type == xhtml.ElementNode && predicate(node) {
			nodes = append(nodes, node)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	return nodes
}

func nodeHasHiddenWebFetchAttrs(node *xhtml.Node) bool {
	if node == nil {
		return false
	}
	for _, attribute := range node.Attr {
		if strings.EqualFold(attribute.Key, "hidden") {
			return true
		}
		if strings.EqualFold(attribute.Key, "aria-hidden") && strings.EqualFold(strings.TrimSpace(attribute.Val), "true") {
			return true
		}
	}
	return false
}

func webFetchAttr(node *xhtml.Node, key string) string {
	if node == nil {
		return ""
	}
	for _, attribute := range node.Attr {
		if strings.EqualFold(attribute.Key, key) {
			return attribute.Val
		}
	}
	return ""
}

func containsAnyWebFetchHint(text string, hints []string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	for _, hint := range hints {
		if strings.Contains(text, hint) {
			return true
		}
	}
	return false
}

func webFetchTextMetrics(node *xhtml.Node, insideLink bool) (int, int) {
	if node == nil {
		return 0, 0
	}
	switch node.Type {
	case xhtml.TextNode:
		textLength := len(strings.Join(strings.Fields(node.Data), " "))
		if insideLink {
			return textLength, textLength
		}
		return textLength, 0
	case xhtml.ElementNode:
		if _, ok := webFetchAlwaysDropTags[strings.ToLower(node.Data)]; ok {
			return 0, 0
		}
	}

	childInsideLink := insideLink || (node.Type == xhtml.ElementNode && node.Data == "a")
	visibleText := 0
	linkText := 0
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		childVisibleText, childLinkText := webFetchTextMetrics(child, childInsideLink)
		visibleText += childVisibleText
		linkText += childLinkText
	}
	return visibleText, linkText
}

func extractRelevantPassages(markdown string, prompt string) []string {
	sections := splitWebFetchSections(markdown)
	if len(sections) == 0 {
		return nil
	}

	keywords := keywordSet(prompt)
	if len(keywords) == 0 {
		return firstNonEmptySections(sections, 3)
	}

	type scoredSection struct {
		text  string
		score int
	}

	scored := make([]scoredSection, 0, len(sections))
	for _, section := range sections {
		sectionKeywords := keywordSet(section)
		score := 0
		for keyword := range keywords {
			if _, ok := sectionKeywords[keyword]; ok {
				score++
			}
		}
		if score > 0 {
			scored = append(scored, scoredSection{text: section, score: score})
		}
	}

	if len(scored) == 0 {
		return firstNonEmptySections(sections, 3)
	}

	for left := 0; left < len(scored)-1; left++ {
		for right := left + 1; right < len(scored); right++ {
			if scored[right].score > scored[left].score {
				scored[left], scored[right] = scored[right], scored[left]
			}
		}
	}

	result := make([]string, 0, min(3, len(scored)))
	for _, item := range scored {
		result = append(result, item.text)
		if len(result) == 3 {
			break
		}
	}
	return result
}

func splitWebFetchSections(markdown string) []string {
	rawSections := strings.FieldsFunc(markdown, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
	sections := make([]string, 0, len(rawSections))
	for _, section := range rawSections {
		section = strings.Join(strings.Fields(section), " ")
		if section != "" {
			sections = append(sections, section)
		}
	}
	return sections
}

func firstNonEmptySections(sections []string, limit int) []string {
	result := make([]string, 0, min(limit, len(sections)))
	for _, section := range sections {
		if strings.TrimSpace(section) == "" {
			continue
		}
		result = append(result, section)
		if len(result) == limit {
			break
		}
	}
	return result
}

func keywordSet(text string) map[string]struct{} {
	text = strings.ToLower(text)
	parts := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	keywords := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		if len(part) < 3 {
			continue
		}
		keywords[part] = struct{}{}
	}
	return keywords
}

func normalizeWebFetchURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("web_fetch requires url")
	}
	if len(rawURL) > maxWebFetchURLLength {
		return "", fmt.Errorf("web_fetch url exceeds %d characters", maxWebFetchURLLength)
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid url %q: %w", rawURL, err)
	}
	if parsed.Scheme == "" {
		return "", fmt.Errorf("web_fetch requires an absolute url")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("web_fetch only supports http and https urls")
	}
	if parsed.User != nil {
		return "", fmt.Errorf("web_fetch does not allow credentials in urls")
	}
	if parsed.Hostname() == "" {
		return "", fmt.Errorf("web_fetch requires a hostname")
	}
	if err := validateWebFetchHost(parsed.Hostname()); err != nil {
		return "", err
	}
	if parsed.Scheme == "http" {
		parsed.Scheme = "https"
		return parsed.String(), nil
	}
	return parsed.String(), nil
}

func validateWebFetchHost(host string) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("web_fetch requires a hostname")
	}

	if ip, err := netip.ParseAddr(host); err == nil {
		if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsMulticast() {
			return fmt.Errorf("web_fetch blocks private or local addresses")
		}
		return nil
	}

	if strings.EqualFold(host, "localhost") || !strings.Contains(host, ".") {
		return fmt.Errorf("web_fetch requires a public hostname")
	}

	addrs, err := net.DefaultResolver.LookupNetIP(context.Background(), "ip", host)
	if err != nil {
		return fmt.Errorf("resolve host %q: %w", host, err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("web_fetch could not resolve %q", host)
	}
	for _, addr := range addrs {
		if !addr.IsGlobalUnicast() || addr.IsPrivate() || addr.IsLoopback() || addr.IsLinkLocalUnicast() || addr.IsMulticast() {
			return fmt.Errorf("web_fetch blocks private or local addresses")
		}
	}
	return nil
}

func isRedirectStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}

func resolveWebFetchRedirect(baseURL, location string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse redirect base url: %w", err)
	}
	redirectURL, err := url.Parse(location)
	if err != nil {
		return "", fmt.Errorf("parse redirect url: %w", err)
	}
	return base.ResolveReference(redirectURL).String(), nil
}

func webFetchPermittedRedirect(originalURL, redirectURL string) bool {
	original, err := url.Parse(originalURL)
	if err != nil {
		return false
	}
	redirected, err := url.Parse(redirectURL)
	if err != nil {
		return false
	}
	if redirected.Scheme != original.Scheme {
		return false
	}
	if redirected.Port() != original.Port() {
		return false
	}
	if redirected.User != nil {
		return false
	}
	return stripWww(original.Hostname()) == stripWww(redirected.Hostname())
}

func stripWww(host string) string {
	return strings.TrimPrefix(strings.ToLower(host), "www.")
}

func looksLikeHTML(body string) bool {
	body = strings.ToLower(strings.TrimSpace(body))
	return strings.Contains(body, "<html") || strings.Contains(body, "<!doctype html") || strings.Contains(body, "<body")
}

type webFetchCache struct {
	mu        sync.Mutex
	maxBytes  int64
	ttl       time.Duration
	usedBytes int64
	entries   map[string]*list.Element
	recency   *list.List
}

type webFetchCacheEntry struct {
	key       string
	value     webFetchContent
	size      int64
	expiresAt time.Time
}

func newWebFetchCache(maxBytes int64, ttl time.Duration) *webFetchCache {
	return &webFetchCache{
		maxBytes: maxBytes,
		ttl:      ttl,
		entries:  make(map[string]*list.Element),
		recency:  list.New(),
	}
}

func (c *webFetchCache) Get(key string) (webFetchContent, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, ok := c.entries[key]
	if !ok {
		return webFetchContent{}, false
	}
	entry := element.Value.(*webFetchCacheEntry)
	if time.Now().After(entry.expiresAt) {
		c.removeElement(element)
		return webFetchContent{}, false
	}
	c.recency.MoveToFront(element)
	return entry.value, true
}

func (c *webFetchCache) Set(key string, value webFetchContent) {
	size := int64(len(value.Markdown))
	if size <= 0 {
		size = 1
	}
	if size > c.maxBytes {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if existing, ok := c.entries[key]; ok {
		c.removeElement(existing)
	}

	entry := &webFetchCacheEntry{
		key:       key,
		value:     value,
		size:      size,
		expiresAt: time.Now().Add(c.ttl),
	}
	element := c.recency.PushFront(entry)
	c.entries[key] = element
	c.usedBytes += size

	for c.usedBytes > c.maxBytes {
		oldest := c.recency.Back()
		if oldest == nil {
			break
		}
		c.removeElement(oldest)
	}
}

func (c *webFetchCache) removeElement(element *list.Element) {
	entry := element.Value.(*webFetchCacheEntry)
	delete(c.entries, entry.key)
	c.recency.Remove(element)
	c.usedBytes -= entry.size
	if c.usedBytes < 0 {
		c.usedBytes = 0
	}
}
