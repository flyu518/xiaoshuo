package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx        context.Context
	configDir  string
	rulesPath  string
	novelsPath string
	cacheMu    sync.Mutex
}

type Novel struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	CatalogURL string `json:"catalogUrl"`
	RuleID     string `json:"ruleId"`
}

type SiteRule struct {
	ID                         string            `json:"id"`
	Name                       string            `json:"name"`
	MatchDomains               []string          `json:"matchDomains"`
	CatalogSectionHeadingText  string            `json:"catalogSectionHeadingText"`
	CatalogSectionContainer    string            `json:"catalogSectionContainer"`
	CatalogChapterLinkSelector string            `json:"catalogChapterLinkSelector"`
	ChapterTitleSelector       string            `json:"chapterTitleSelector"`
	ChapterContentSelector     string            `json:"chapterContentSelector"`
	NextPageSelector           string            `json:"nextPageSelector"`
	NextChapterSelector        string            `json:"nextChapterSelector"`
	ContentCleanupSelectors    []string          `json:"contentCleanupSelectors"`
	ContentStopTexts           []string          `json:"contentStopTexts"`
	RemoveMatchingLines        []string          `json:"removeMatchingLines"`
	TextReplacementRules       []TextReplacementRule `json:"textReplacementRules"`
	RegexReplacementRules      []RegexReplacementRule `json:"regexReplacementRules"`
	SkipChapterTitlePatterns   []string          `json:"skipChapterTitlePatterns"`
	RequestHeaders             map[string]string `json:"requestHeaders"`
	Notes                      string            `json:"notes"`
}

type TextReplacementRule struct {
	Match         string `json:"match"`
	Replace       string `json:"replace"`
	CaseSensitive bool   `json:"caseSensitive"`
	ReplaceFirst  bool   `json:"replaceFirst"`
	Enabled       bool   `json:"enabled"`
}

type RegexReplacementRule struct {
	Pattern       string `json:"pattern"`
	Replace       string `json:"replace"`
	RemoveLine    bool   `json:"removeLine"`
	ReplaceFirst  bool   `json:"replaceFirst"`
	Enabled       bool   `json:"enabled"`
}

type AppState struct {
	Rules  []SiteRule `json:"rules"`
	Novels []Novel    `json:"novels"`
}

type CatalogRequest struct {
	CatalogURL string `json:"catalogUrl"`
	RuleID     string `json:"ruleId"`
	NovelID    string `json:"novelId"`
}

type CatalogChapter struct {
	Title  string `json:"title"`
	URL    string `json:"url"`
	Cached bool   `json:"cached"`
}

type CatalogAnalysis struct {
	RuleID       string           `json:"ruleId"`
	RuleName     string           `json:"ruleName"`
	NovelTitle   string           `json:"novelTitle"`
	ChapterCount int              `json:"chapterCount"`
	Chapters     []CatalogChapter `json:"chapters"`
}

type ExportRequest struct {
	CatalogURL          string   `json:"catalogUrl"`
	RuleID              string   `json:"ruleId"`
	NovelTitle          string   `json:"novelTitle"`
	NovelID             string   `json:"novelId"`
	SelectedChapterURLs []string `json:"selectedChapterUrls"`
	StartChapter        int      `json:"startChapter"`
	EndChapter          int      `json:"endChapter"`
	MaxChapters         int      `json:"maxChapters"`
	RetryCount          int      `json:"retryCount"`
	SkipOnFailure       bool     `json:"skipOnFailure"`
	SkipFilteredTitle   bool     `json:"skipFilteredTitle"`
}

type ChapterReadRequest struct {
	CatalogURL   string `json:"catalogUrl"`
	RuleID       string `json:"ruleId"`
	NovelID      string `json:"novelId"`
	ChapterURL   string `json:"chapterUrl"`
	ChapterTitle string `json:"chapterTitle"`
}

type ChapterReadResult struct {
	RuleID       string `json:"ruleId"`
	NovelTitle   string `json:"novelTitle"`
	ChapterTitle string `json:"chapterTitle"`
	ChapterURL   string `json:"chapterUrl"`
	Content      string `json:"content"`
	Cached       bool   `json:"cached"`
}

type ExportFailure struct {
	Index   int    `json:"index"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Error   string `json:"error"`
	Retries int    `json:"retries"`
}

type ExportResult struct {
	FilePath      string          `json:"filePath"`
	RuleID        string          `json:"ruleId"`
	NovelTitle    string          `json:"novelTitle"`
	ExportedCount int             `json:"exportedCount"`
	FailureCount  int             `json:"failureCount"`
	Failures      []ExportFailure `json:"failures"`
}

type ProgressEvent struct {
	Stage        string `json:"stage"`
	Message      string `json:"message"`
	Current      int    `json:"current"`
	Total        int    `json:"total"`
	ChapterTitle string `json:"chapterTitle,omitempty"`
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}

	a.configDir = filepath.Join(configDir, "xiaoshuo")
	a.rulesPath = filepath.Join(a.configDir, "rules.json")
	a.novelsPath = filepath.Join(a.configDir, "novels.json")

	if err := os.MkdirAll(a.configDir, 0o755); err != nil {
		runtime.LogErrorf(a.ctx, "create config dir failed: %v", err)
		return
	}
	if err := a.ensureRulesFile(); err != nil {
		runtime.LogErrorf(a.ctx, "init rules failed: %v", err)
		return
	}
	if err := a.ensureNovelsFile(); err != nil {
		runtime.LogErrorf(a.ctx, "init novels failed: %v", err)
		return
	}
	if err := a.persistNormalizedRules(); err != nil {
		runtime.LogErrorf(a.ctx, "normalize rules failed: %v", err)
	}
}

func (a *App) LoadState() (*AppState, error) {
	rules, err := a.loadRules()
	if err != nil {
		return nil, err
	}
	novels, err := a.loadNovels()
	if err != nil {
		return nil, err
	}
	return &AppState{Rules: rules, Novels: novels}, nil
}

func (a *App) SaveRule(rule SiteRule) (*AppState, error) {
	rules, err := a.loadRules()
	if err != nil {
		return nil, err
	}
	novels, err := a.loadNovels()
	if err != nil {
		return nil, err
	}

	rule = normalizeRule(rule)
	if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule-%d", time.Now().UnixNano())
	}

	index := slices.IndexFunc(rules, func(item SiteRule) bool { return item.ID == rule.ID })
	if index >= 0 {
		rules[index] = rule
	} else {
		rules = append(rules, rule)
	}

	if err := a.saveRules(rules); err != nil {
		return nil, err
	}
	if index >= 0 {
		if err := a.clearRuleCaches(rule.ID); err != nil {
			return nil, err
		}
	}
	return &AppState{Rules: rules, Novels: novels}, nil
}

func (a *App) DeleteRule(ruleID string) (*AppState, error) {
	rules, err := a.loadRules()
	if err != nil {
		return nil, err
	}
	novels, err := a.loadNovels()
	if err != nil {
		return nil, err
	}

	filtered := make([]SiteRule, 0, len(rules))
	for _, rule := range rules {
		if rule.ID != ruleID {
			filtered = append(filtered, rule)
		}
	}
	if len(filtered) == len(rules) {
		return nil, errors.New("rule not found")
	}
	if err := a.saveRules(filtered); err != nil {
		return nil, err
	}
	return &AppState{Rules: filtered, Novels: novels}, nil
}

func (a *App) AnalyzeCatalog(req CatalogRequest) (*CatalogAnalysis, error) {
	if req.NovelID != "" {
		novel, rule, err := a.resolveNovel(req.NovelID)
		if err != nil {
			return nil, err
		}
		analysis, err := a.analyzeCatalog(rule, novel.CatalogURL)
		if err != nil {
			return nil, err
		}
		if err := a.markCachedChapters(analysis, novel.ID, rule.ID); err != nil {
			return nil, err
		}
		return analysis, nil
	}

	rule, err := a.resolveRule(req.RuleID, req.CatalogURL)
	if err != nil {
		return nil, err
	}
	if req.RuleID != "" && !ruleMatchesURL(*rule, req.CatalogURL) {
		return nil, errors.New("当前目录地址与所选规则不匹配，请切换规则或更换目录地址")
	}
	return a.analyzeCatalog(rule, req.CatalogURL)
}

func (a *App) ReadChapter(req ChapterReadRequest) (*ChapterReadResult, error) {
	chapterURL := strings.TrimSpace(req.ChapterURL)
	if chapterURL == "" {
		return nil, errors.New("chapter URL is required")
	}

	chapter := CatalogChapter{
		Title: strings.TrimSpace(req.ChapterTitle),
		URL:   chapterURL,
	}

	var (
		rule       *SiteRule
		novelTitle string
		err        error
	)

	if strings.TrimSpace(req.NovelID) != "" {
		novel, resolvedRule, err := a.resolveNovel(req.NovelID)
		if err != nil {
			return nil, err
		}
		rule = resolvedRule
		novelTitle = novel.Title
		if content, found, err := a.readChapterCache(rule.ID, chapter.URL); err != nil {
			return nil, err
		} else if found && strings.TrimSpace(content) != "" {
			return &ChapterReadResult{
				RuleID:       rule.ID,
				NovelTitle:   novelTitle,
				ChapterTitle: chapter.Title,
				ChapterURL:   chapter.URL,
				Content:      content,
				Cached:       true,
			}, nil
		}
	} else {
		if ruleID := strings.TrimSpace(req.RuleID); ruleID != "" {
			if content, found, err := a.readChapterCache(ruleID, chapter.URL); err != nil {
				return nil, err
			} else if found && strings.TrimSpace(content) != "" {
				return &ChapterReadResult{
					RuleID:       ruleID,
					NovelTitle:   novelTitle,
					ChapterTitle: chapter.Title,
					ChapterURL:   chapter.URL,
					Content:      content,
					Cached:       true,
				}, nil
			}
		}

		rule, err = a.resolveRule(req.RuleID, req.CatalogURL)
		if err != nil {
			return nil, err
		}
	}

	content, _, cached, err := a.extractChapterWithCache(rule, chapter, 0)
	if err != nil {
		return nil, err
	}

	return &ChapterReadResult{
		RuleID:       rule.ID,
		NovelTitle:   novelTitle,
		ChapterTitle: chapter.Title,
		ChapterURL:   chapter.URL,
		Content:      content,
		Cached:       cached,
	}, nil
}

func (a *App) markCachedChapters(analysis *CatalogAnalysis, novelID, ruleID string) error {
	if analysis == nil || strings.TrimSpace(novelID) == "" {
		return nil
	}

	cachedURLs, err := a.cachedChapterURLsForNovel(novelID, ruleID)
	if err != nil {
		return err
	}
	for index, chapter := range analysis.Chapters {
		_, analysis.Chapters[index].Cached = cachedURLs[chapter.URL]
	}
	return nil
}

func (a *App) ExportNovel(req ExportRequest) (*ExportResult, error) {
	if req.NovelID != "" {
		novel, _, err := a.resolveNovel(req.NovelID)
		if err != nil {
			return nil, err
		}
		req.CatalogURL = novel.CatalogURL
		req.RuleID = novel.RuleID
		if strings.TrimSpace(req.NovelTitle) == "" {
			req.NovelTitle = novel.Title
		}
	}

	rule, err := a.resolveRule(req.RuleID, req.CatalogURL)
	if err != nil {
		return nil, err
	}
	analysis, err := a.analyzeCatalog(rule, req.CatalogURL)
	if err != nil {
		return nil, err
	}

	novelTitle := strings.TrimSpace(req.NovelTitle)
	if novelTitle == "" {
		novelTitle = analysis.NovelTitle
	}
	if novelTitle == "" {
		novelTitle = "Novel"
	}

	chapters := analysis.Chapters
	if req.SkipFilteredTitle {
		chapters = filterChapters(chapters, rule.SkipChapterTitlePatterns)
	}
	if req.MaxChapters > 0 && req.MaxChapters < len(chapters) {
		chapters = chapters[:req.MaxChapters]
	}
	if len(chapters) == 0 {
		return nil, errors.New("no chapters available for export")
	}

	defaultName := sanitizeFilename(novelTitle)
	if defaultName == "" {
		defaultName = "novel"
	}
	defaultName += ".txt"

	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Choose TXT export path",
		DefaultFilename: defaultName,
		Filters:         []runtime.FileFilter{{DisplayName: "Text Document (*.txt)", Pattern: "*.txt"}},
	})
	if err != nil {
		return nil, err
	}
	if filePath == "" {
		return nil, errors.New("export cancelled")
	}

	a.emitProgress(ProgressEvent{Stage: "start", Message: "Starting extraction", Current: 0, Total: len(chapters)})

	var builder strings.Builder
	builder.WriteString(novelTitle)
	builder.WriteString("\n\n")
	for index, chapter := range chapters {
		a.emitProgress(ProgressEvent{Stage: "chapter", Message: fmt.Sprintf("Fetching chapter %d/%d", index+1, len(chapters)), Current: index + 1, Total: len(chapters), ChapterTitle: chapter.Title})
		text, err := a.extractChapter(rule, chapter)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch chapter %s: %w", chapter.Title, err)
		}
		builder.WriteString(chapter.Title)
		builder.WriteString("\n\n")
		builder.WriteString(text)
		builder.WriteString("\n\n")
	}

	if err := os.WriteFile(filePath, []byte(strings.TrimSpace(builder.String())+"\n"), 0o644); err != nil {
		return nil, err
	}
	return &ExportResult{FilePath: filePath, RuleID: rule.ID, NovelTitle: novelTitle, ExportedCount: len(chapters)}, nil
}

func (a *App) ensureRulesFile() error {
	if _, err := os.Stat(a.rulesPath); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return a.saveRules(defaultRules())
}

func (a *App) loadRules() ([]SiteRule, error) {
	if err := a.ensureRulesFile(); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(a.rulesPath)
	if err != nil {
		return nil, err
	}
	var rules []SiteRule
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, err
	}
	for index, rule := range rules {
		rules[index] = normalizeRule(rule)
	}
	return rules, nil
}

func (a *App) persistNormalizedRules() error {
	rules, err := a.loadRules()
	if err != nil {
		return err
	}
	return a.saveRules(rules)
}

func (a *App) saveRules(rules []SiteRule) error {
	for index, rule := range rules {
		rules[index] = normalizeRule(rule)
	}
	data, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.rulesPath, data, 0o644)
}

func (a *App) resolveRule(ruleID, rawURL string) (*SiteRule, error) {
	rules, err := a.loadRules()
	if err != nil {
		return nil, err
	}
	if ruleID != "" {
		for _, rule := range rules {
			if rule.ID == ruleID {
				copied := rule
				return &copied, nil
			}
		}
		return nil, errors.New("specified rule was not found")
	}
	for _, rule := range rules {
		if ruleMatchesURL(rule, rawURL) {
			copied := rule
			return &copied, nil
		}
	}
	return nil, errors.New("no matching site rule found; please choose or create one")
}

func ruleMatchesURL(rule SiteRule, rawURL string) bool {
	target := strings.ToLower(rawURL)
	for _, domain := range rule.MatchDomains {
		domain = strings.ToLower(strings.TrimSpace(domain))
		if domain != "" && strings.Contains(target, domain) {
			return true
		}
	}
	return false
}

func normalizeRule(rule SiteRule) SiteRule {
	rule.Name = strings.TrimSpace(rule.Name)
	rule.CatalogSectionHeadingText = strings.TrimSpace(rule.CatalogSectionHeadingText)
	rule.CatalogSectionContainer = strings.TrimSpace(rule.CatalogSectionContainer)
	rule.CatalogChapterLinkSelector = strings.TrimSpace(rule.CatalogChapterLinkSelector)
	rule.ChapterTitleSelector = strings.TrimSpace(rule.ChapterTitleSelector)
	rule.ChapterContentSelector = strings.TrimSpace(rule.ChapterContentSelector)
	rule.NextPageSelector = strings.TrimSpace(rule.NextPageSelector)
	rule.NextChapterSelector = strings.TrimSpace(rule.NextChapterSelector)
	rule.Notes = strings.TrimSpace(rule.Notes)
	if rule.RequestHeaders == nil {
		rule.RequestHeaders = map[string]string{}
	}
	rule.MatchDomains = compactStrings(rule.MatchDomains)
	rule.ContentCleanupSelectors = compactStrings(rule.ContentCleanupSelectors)
	rule.ContentStopTexts = compactStrings(rule.ContentStopTexts)
	rule.RemoveMatchingLines = compactStrings(rule.RemoveMatchingLines)
	rule.TextReplacementRules = normalizeTextReplacementRules(rule.TextReplacementRules)
	rule.RegexReplacementRules = normalizeRegexReplacementRules(rule.RegexReplacementRules)
	rule.SkipChapterTitlePatterns = compactStrings(rule.SkipChapterTitlePatterns)
	return repairKnownRule(rule)
}

func normalizeTextReplacementRules(rules []TextReplacementRule) []TextReplacementRule {
	result := make([]TextReplacementRule, 0, len(rules))
	for _, rule := range rules {
		rule.Match = strings.TrimSpace(rule.Match)
		rule.Replace = strings.TrimSpace(rule.Replace)
		if rule.Match == "" {
			continue
		}
		result = append(result, rule)
	}
	return result
}

func normalizeRegexReplacementRules(rules []RegexReplacementRule) []RegexReplacementRule {
	result := make([]RegexReplacementRule, 0, len(rules))
	for _, rule := range rules {
		rule.Pattern = strings.TrimSpace(rule.Pattern)
		rule.Replace = strings.TrimSpace(rule.Replace)
		if rule.Pattern == "" {
			continue
		}
		result = append(result, rule)
	}
	return result
}

func repairKnownRule(rule SiteRule) SiteRule {
	switch rule.ID {
	case "zhswx":
		rule.Name = "宙斯小说网"
		rule.MatchDomains = []string{"zhswx.com"}
		rule.CatalogChapterLinkSelector = "td.chapterlist a[href^='/read/']"
		rule.ChapterContentSelector = "div[style*='font-size: 20px'][style*='width: 700px']"
		rule.ContentStopTexts = []string{"手机阅读请访问", "温馨提示：按 回车[Enter]键 返回书目"}
		rule.SkipChapterTitlePatterns = []string{"请假", "总结", "感言", "生日快乐", "深夜聊点什么"}
	case "23shuku":
		rule.Name = "二三书库 / 笔趣阁镜像"
		rule.MatchDomains = []string{"23.225.162.18", "23txxt.com"}
		rule.CatalogChapterLinkSelector = "h2.layout-tit:contains('正文') + div.section-box ul.section-list a[href*='/bqg/']"
		rule.ChapterContentSelector = "#content"
		rule.NextPageSelector = "a:contains('下一页')"
		rule.NextChapterSelector = "a:contains('下一章')"
		rule.ContentCleanupSelectors = []string{"script"}
		rule.ContentStopTexts = []string{"（本章未完，请点击下一页继续阅读）"}
		rule.SkipChapterTitlePatterns = []string{"请假", "总结", "感言", "生日快乐", "今日无更", "晚点发", "住院", "新年快乐"}
	}
	return rule
}

func compactStrings(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func filterChapters(chapters []CatalogChapter, patterns []string) []CatalogChapter {
	if len(patterns) == 0 {
		return chapters
	}
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err == nil {
			compiled = append(compiled, re)
		}
	}
	if len(compiled) == 0 {
		return chapters
	}
	filtered := make([]CatalogChapter, 0, len(chapters))
	for _, chapter := range chapters {
		skip := false
		for _, re := range compiled {
			if re.MatchString(chapter.Title) {
				skip = true
				break
			}
		}
		if !skip {
			filtered = append(filtered, chapter)
		}
	}
	return filtered
}

func filterChaptersBySelection(chapters []CatalogChapter, selectedURLs []string) ([]CatalogChapter, error) {
	if len(selectedURLs) == 0 {
		return chapters, nil
	}

	selected := make(map[string]struct{}, len(selectedURLs))
	for _, rawURL := range selectedURLs {
		rawURL = strings.TrimSpace(rawURL)
		if rawURL != "" {
			selected[rawURL] = struct{}{}
		}
	}

	filtered := make([]CatalogChapter, 0, len(chapters))
	for _, chapter := range chapters {
		if _, ok := selected[chapter.URL]; ok {
			filtered = append(filtered, chapter)
		}
	}
	if len(filtered) == 0 {
		return nil, errors.New("no chapters selected for export")
	}
	return filtered, nil
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer("<", "", ">", "", ":", " ", "\"", "", "/", "-", "\\", "-", "|", "-", "?", "", "*", "")
	name = replacer.Replace(strings.TrimSpace(name))
	name = strings.Join(strings.Fields(name), " ")
	return strings.TrimSpace(name)
}

func (a *App) emitProgress(event ProgressEvent) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "crawl-progress", event)
	}
}

func defaultRules() []SiteRule {
	return []SiteRule{
		{ID: "zhswx", Notes: "Catalog uses td.chapterlist; chapter content lives in an inline-style div."},
		{ID: "23shuku", Notes: "This site may split one chapter into multiple pages. Finish next-page pagination before moving on."},
	}
}
