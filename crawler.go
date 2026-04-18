package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"path"
	"regexp"
	stdruntime "runtime"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

func (a *App) analyzeCatalog(rule *SiteRule, rawURL string) (*CatalogAnalysis, error) {
	doc, baseURL, err := a.fetchDocument(rawURL, rule.RequestHeaders)
	if err != nil {
		return nil, err
	}

	chapters, err := extractCatalogChapters(doc, baseURL, *rule)
	if err != nil {
		return nil, err
	}

	novelTitle := strings.TrimSpace(doc.Find("h1").First().Text())
	if novelTitle == "" {
		title := cleanInlineText(doc.Find("title").First().Text())
		novelTitle = cleanupNovelTitle(title)
	}
	if novelTitle == "" {
		novelTitle = "Novel"
	}

	return &CatalogAnalysis{
		RuleID:       rule.ID,
		RuleName:     rule.Name,
		NovelTitle:   novelTitle,
		ChapterCount: len(chapters),
		Chapters:     chapters,
	}, nil
}

func extractCatalogChapters(doc *goquery.Document, baseURL *url.URL, rule SiteRule) ([]CatalogChapter, error) {
	selection := doc.Selection

	if rule.CatalogSectionHeadingText != "" {
		var target *goquery.Selection
		doc.Find("h1, h2, h3, h4, h5").EachWithBreak(func(_ int, s *goquery.Selection) bool {
			if cleanInlineText(s.Text()) == rule.CatalogSectionHeadingText {
				target = s
				return false
			}
			return true
		})

		if target != nil && rule.CatalogSectionContainer != "" {
			selection = target.NextAllFiltered(rule.CatalogSectionContainer).First()
		}
	}

	if rule.CatalogChapterLinkSelector == "" {
		return nil, errors.New("rule is missing a catalog chapter selector")
	}

	result := make([]CatalogChapter, 0)
	seen := map[string]bool{}

	selection.Find(rule.CatalogChapterLinkSelector).Each(func(_ int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok {
			return
		}

		resolved := resolveURL(baseURL, href)
		if resolved == "" || seen[resolved] {
			return
		}

		title := cleanInlineText(s.Text())
		if title == "" {
			return
		}

		seen[resolved] = true
		result = append(result, CatalogChapter{
			Title: title,
			URL:   resolved,
		})
	})

	if len(result) == 0 {
		return nil, errors.New("no chapters were found on the catalog page")
	}

	return result, nil
}

func (a *App) extractChapter(rule *SiteRule, chapter CatalogChapter) (string, error) {
	if rule.ChapterContentSelector == "" {
		return "", errors.New("rule is missing a chapter content selector")
	}

	var pages []string
	currentURL := chapter.URL
	visited := map[string]bool{}
	baseChapterPath := chapterPathKey(chapter.URL)

	for {
		if visited[currentURL] {
			break
		}
		visited[currentURL] = true

		doc, baseURL, err := a.fetchDocument(currentURL, rule.RequestHeaders)
		if err != nil {
			return "", err
		}

		content := doc.Find(rule.ChapterContentSelector).First()
		if content.Length() == 0 {
			return "", fmt.Errorf("chapter content node not found: %s", rule.ChapterContentSelector)
		}

		for _, cleanupSelector := range rule.ContentCleanupSelectors {
			content.Find(cleanupSelector).Remove()
		}

		htmlText, err := content.Html()
		if err != nil {
			return "", err
		}

		pageText := htmlToText(htmlText)
		pageText = stripStopTexts(pageText, rule.ContentStopTexts)
		pageText = applyTextCleanupRules(pageText, *rule)
		pageText = removePageTitleNoise(pageText, chapter.Title)
		if pageText != "" {
			pages = append(pages, pageText)
		}

		nextPageURL := ""
		if rule.NextPageSelector != "" {
			nextPageURL = resolveURL(baseURL, content.Parent().Find(rule.NextPageSelector).First().AttrOr("href", ""))
			if nextPageURL == "" {
				nextPageURL = resolveURL(baseURL, doc.Find(rule.NextPageSelector).First().AttrOr("href", ""))
			}
		}

		if nextPageURL == "" || visited[nextPageURL] || chapterPathKey(nextPageURL) != baseChapterPath {
			break
		}

		currentURL = nextPageURL
	}

	text := strings.TrimSpace(strings.Join(pages, "\n\n"))
	text = cleanupRepeatedBlankLines(text)
	if text == "" {
		return "", errors.New("extracted chapter text is empty")
	}
	return text, nil
}

func (a *App) extractChapterWithRetry(rule *SiteRule, chapter CatalogChapter, retryCount int) (string, int, error) {
	if retryCount < 0 {
		retryCount = 0
	}

	var lastErr error
	for attempt := 0; attempt <= retryCount; attempt++ {
		text, err := a.extractChapter(rule, chapter)
		if err == nil {
			return text, attempt, nil
		}
		lastErr = err
	}

	return "", retryCount, lastErr
}

func (a *App) fetchDocument(rawURL string, headers map[string]string) (*goquery.Document, *url.URL, error) {
	rawURL = strings.TrimSpace(rawURL)

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, nil, err
	}

	resp, err := doHTTPRequest(rawURL, headers, false)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()

		// Some novel sites are picky about HTTP stack details. Retry with a
		// more browser-like, HTTP/1.1-only transport before giving up.
		resp, err = doHTTPRequest(rawURL, headers, true)
		if err != nil {
			return a.fetchDocumentWithFallback(rawURL, parsed, headers)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			defer resp.Body.Close()
			return a.fetchDocumentWithFallback(rawURL, parsed, headers)
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	utf8Body, err := decodeHTML(resp.Header.Get("Content-Type"), body)
	if err != nil {
		return nil, nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(utf8Body))
	if err != nil {
		return nil, nil, err
	}

	return doc, parsed, nil
}

func (a *App) fetchDocumentWithFallback(rawURL string, parsed *url.URL, headers map[string]string) (*goquery.Document, *url.URL, error) {
	var body []byte
	var err error

	body, err = fetchBodyWithCurl(rawURL, headers)
	if err != nil && stdruntime.GOOS == "windows" {
		body, err = fetchBodyWithPowerShell(rawURL, headers)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("request failed for %s: 404 Not Found", rawURL)
	}

	utf8Body, err := decodeHTML("", body)
	if err != nil {
		return nil, nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(utf8Body))
	if err != nil {
		return nil, nil, err
	}

	return doc, parsed, nil
}

func doHTTPRequest(rawURL string, headers map[string]string, conservative bool) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 25 * time.Second}
	if conservative {
		client.Transport = &http.Transport{
			ForceAttemptHTTP2:     false,
			DisableCompression:    false,
			MaxIdleConns:          10,
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: 20 * time.Second,
		}
	}

	return client.Do(req)
}

func fetchBodyWithCurl(rawURL string, headers map[string]string) ([]byte, error) {
	if _, err := exec.LookPath("curl"); err != nil {
		return nil, err
	}

	args := []string{
		"-L",
		"-f",
		"-sS",
		"--compressed",
		"-A", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36",
		"-H", "Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
		"-H", "Accept-Language: zh-CN,zh;q=0.9",
		"-H", "Cache-Control: no-cache",
		"-H", "Pragma: no-cache",
		"-H", "Upgrade-Insecure-Requests: 1",
	}

	for key, value := range headers {
		args = append(args, "-H", fmt.Sprintf("%s: %s", key, value))
	}

	args = append(args, rawURL)

	cmd := exec.Command("curl", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	if len(output) == 0 {
		return nil, errors.New("empty curl response")
	}

	return output, nil
}

func fetchBodyWithPowerShell(rawURL string, headers map[string]string) ([]byte, error) {
	if _, err := exec.LookPath("powershell"); err != nil {
		return nil, err
	}

	headerLines := []string{
		"'User-Agent'='Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36'",
		"'Accept'='text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8'",
		"'Accept-Language'='zh-CN,zh;q=0.9'",
		"'Cache-Control'='no-cache'",
		"'Pragma'='no-cache'",
	}

	for key, value := range headers {
		safeKey := strings.ReplaceAll(key, "'", "''")
		safeValue := strings.ReplaceAll(value, "'", "''")
		headerLines = append(headerLines, fmt.Sprintf("'%s'='%s'", safeKey, safeValue))
	}

	safeURL := strings.ReplaceAll(rawURL, "'", "''")
	script := fmt.Sprintf(
		"[Console]::OutputEncoding = [System.Text.Encoding]::UTF8; $ProgressPreference='SilentlyContinue'; $headers=@{%s}; (Invoke-WebRequest -UseBasicParsing -Headers $headers '%s').Content",
		strings.Join(headerLines, "; "),
		safeURL,
	)

	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	if len(output) == 0 {
		return nil, errors.New("empty powershell response")
	}

	return output, nil
}

func decodeHTML(contentType string, body []byte) ([]byte, error) {
	reader, err := charset.NewReader(bytes.NewReader(body), contentType)
	if err == nil {
		return io.ReadAll(reader)
	}

	reader, err = charset.NewReaderLabel("utf-8", bytes.NewReader(body))
	if err == nil {
		return io.ReadAll(reader)
	}

	return body, nil
}

func resolveURL(baseURL *url.URL, href string) string {
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(strings.ToLower(href), "javascript:") {
		return ""
	}
	ref, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return baseURL.ResolveReference(ref).String()
}

func htmlToText(fragment string) string {
	node, err := html.Parse(strings.NewReader("<div>" + fragment + "</div>"))
	if err != nil {
		return cleanInlineText(fragment)
	}

	var builder strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			builder.WriteString(html.UnescapeString(n.Data))
		}

		if n.Type == html.ElementNode {
			switch n.Data {
			case "br", "p", "div", "li", "section", "article", "tr":
				builder.WriteString("\n")
			}
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}

		if n.Type == html.ElementNode {
			switch n.Data {
			case "p", "div", "li", "section", "article", "tr":
				builder.WriteString("\n")
			}
		}
	}

	walk(node)
	return strings.TrimSpace(cleanupRepeatedBlankLines(builder.String()))
}

func stripStopTexts(text string, stopTexts []string) string {
	for _, stopText := range stopTexts {
		text = strings.ReplaceAll(text, stopText, "")
	}
	return cleanupRepeatedBlankLines(text)
}

func applyTextCleanupRules(text string, rule SiteRule) string {
	text = removeMatchingLines(text, rule.RemoveMatchingLines)
	text = applyTextReplacementRules(text, rule.TextReplacementRules)
	text = applyRegexReplacementRules(text, rule.RegexReplacementRules)
	return cleanupRepeatedBlankLines(text)
}

func removeMatchingLines(text string, patterns []string) string {
	if len(patterns) == 0 || strings.TrimSpace(text) == "" {
		return text
	}

	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		remove := false
		for _, pattern := range patterns {
			pattern = strings.TrimSpace(pattern)
			if pattern != "" && strings.Contains(line, pattern) {
				remove = true
				break
			}
		}
		if !remove {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

func applyTextReplacementRules(text string, rules []TextReplacementRule) string {
	for _, rule := range rules {
		if !rule.Enabled || rule.Match == "" {
			continue
		}

		if rule.CaseSensitive {
			if rule.ReplaceFirst {
				text = strings.Replace(text, rule.Match, rule.Replace, 1)
			} else {
				text = strings.ReplaceAll(text, rule.Match, rule.Replace)
			}
			continue
		}

		text = replaceLiteralInsensitive(text, rule.Match, rule.Replace, rule.ReplaceFirst)
	}
	return text
}

func applyRegexReplacementRules(text string, rules []RegexReplacementRule) string {
	for _, rule := range rules {
		if !rule.Enabled || rule.Pattern == "" {
			continue
		}

		re, err := regexp.Compile(rule.Pattern)
		if err != nil {
			continue
		}

		if rule.RemoveLine {
			lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
			kept := make([]string, 0, len(lines))
			for _, line := range lines {
				if !re.MatchString(line) {
					kept = append(kept, line)
				}
			}
			text = strings.Join(kept, "\n")
			continue
		}

		if rule.ReplaceFirst {
			indexes := re.FindStringIndex(text)
			if indexes != nil {
				text = text[:indexes[0]] + re.ReplaceAllString(text[indexes[0]:indexes[1]], rule.Replace) + text[indexes[1]:]
			}
			continue
		}

		text = re.ReplaceAllString(text, rule.Replace)
	}
	return text
}

func replaceLiteralInsensitive(text, match, replace string, replaceFirst bool) string {
	if match == "" {
		return text
	}

	lowerText := strings.ToLower(text)
	lowerMatch := strings.ToLower(match)
	if replaceFirst {
		index := strings.Index(lowerText, lowerMatch)
		if index < 0 {
			return text
		}
		return text[:index] + replace + text[index+len(match):]
	}

	var builder strings.Builder
	searchStart := 0
	for {
		index := strings.Index(lowerText[searchStart:], lowerMatch)
		if index < 0 {
			builder.WriteString(text[searchStart:])
			break
		}
		actualIndex := searchStart + index
		builder.WriteString(text[searchStart:actualIndex])
		builder.WriteString(replace)
		searchStart = actualIndex + len(match)
	}
	return builder.String()
}

func removePageTitleNoise(text, chapterTitle string) string {
	lines := strings.Split(text, "\n")
	cleaned := make([]string, 0, len(lines))
	pageMarkRE := regexp.MustCompile(`\s*\([^)]*\d+/\d+[^)]*\)$`)

	for index, line := range lines {
		line = strings.TrimSpace(line)
		if index == 0 {
			line = pageMarkRE.ReplaceAllString(line, "")
			if line == chapterTitle {
				continue
			}
		}

		if line != "" {
			cleaned = append(cleaned, line)
		} else {
			cleaned = append(cleaned, "")
		}
	}

	return cleanupRepeatedBlankLines(strings.Join(cleaned, "\n"))
}

func cleanupNovelTitle(title string) string {
	title = strings.ReplaceAll(title, "_", " ")
	for _, separator := range []string{"Latest", "latest", "TXT", "-", "\u6700\u65b0\u7ae0\u8282\u5217\u8868", "\u5168\u6587\u9605\u8bfb", "\u5728\u7ebf\u9605\u8bfb"} {
		if index := strings.Index(title, separator); index > 0 {
			title = title[:index]
		}
	}
	return cleanInlineText(title)
}

func cleanInlineText(input string) string {
	input = html.UnescapeString(input)
	input = strings.ReplaceAll(input, "\u00a0", " ")
	return strings.Join(strings.Fields(strings.TrimSpace(input)), " ")
}

func cleanupRepeatedBlankLines(input string) string {
	lines := strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n")
	result := make([]string, 0, len(lines))
	lastBlank := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if !lastBlank {
				result = append(result, "")
			}
			lastBlank = true
			continue
		}
		result = append(result, line)
		lastBlank = false
	}
	return strings.TrimSpace(strings.Join(result, "\n"))
}

func chapterPathKey(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	base := path.Base(parsed.Path)
	base = strings.TrimSuffix(base, path.Ext(base))
	base = regexp.MustCompile(`_\d+$`).ReplaceAllString(base, "")
	return strings.TrimSuffix(parsed.Path, path.Base(parsed.Path)) + base
}
