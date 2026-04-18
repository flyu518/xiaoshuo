package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSliceChaptersByRange(t *testing.T) {
	chapters := []CatalogChapter{{Title: "1"}, {Title: "2"}, {Title: "3"}, {Title: "4"}}

	sliced, err := sliceChaptersByRange(chapters, 2, 3)
	if err != nil {
		t.Fatalf("sliceChaptersByRange returned error: %v", err)
	}
	if len(sliced) != 2 || sliced[0].Title != "2" || sliced[1].Title != "3" {
		t.Fatalf("unexpected range result: %+v", sliced)
	}
}

func TestPersistNormalizedRules_RewritesKnownRules(t *testing.T) {
	tempDir := t.TempDir()
	rulesPath := filepath.Join(tempDir, "rules.json")

	garbled := []SiteRule{{
		ID:                         "zhswx",
		Name:                       "garbled",
		MatchDomains:               []string{"zhswx.com"},
		CatalogChapterLinkSelector: "td.chapterlist a[href^='/read/']",
		ChapterContentSelector:     "div[style*='font-size: 20px'][style*='width: 700px']",
		ContentStopTexts:           []string{"bad"},
		SkipChapterTitlePatterns:   []string{"bad"},
	}}

	data, err := json.Marshal(garbled)
	if err != nil {
		t.Fatalf("marshal garbled rules failed: %v", err)
	}
	if err := os.WriteFile(rulesPath, data, 0o644); err != nil {
		t.Fatalf("write rules file failed: %v", err)
	}

	app := &App{ctx: context.Background(), configDir: tempDir, rulesPath: rulesPath}
	if err := app.persistNormalizedRules(); err != nil {
		t.Fatalf("persistNormalizedRules returned error: %v", err)
	}

	saved, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatalf("read normalized rules failed: %v", err)
	}

	var rules []SiteRule
	if err := json.Unmarshal(saved, &rules); err != nil {
		t.Fatalf("unmarshal normalized rules failed: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}

	rule := rules[0]
	if rule.Name != "宙斯小说网" {
		t.Fatalf("expected repaired rule name, got %q", rule.Name)
	}
	if len(rule.ContentStopTexts) == 0 || rule.ContentStopTexts[0] != "手机阅读请访问" {
		t.Fatalf("expected repaired content stop texts, got %+v", rule.ContentStopTexts)
	}
}

func TestDefaultNovelsFromRules_MigratesLegacyRuleFields(t *testing.T) {
	tempDir := t.TempDir()
	rulesPath := filepath.Join(tempDir, "rules.json")

	legacyRules := []map[string]any{
		{
			"id":         "zhswx",
			"name":       "宙斯小说网",
			"catalogUrl": "https://www.zhswx.com/chapter/67027.html",
			"novelTitle": "青山",
		},
		{
			"id":   "23shuku",
			"name": "二三书库 / 笔趣阁镜像",
		},
	}

	data, err := json.Marshal(legacyRules)
	if err != nil {
		t.Fatalf("marshal legacy rules failed: %v", err)
	}
	if err := os.WriteFile(rulesPath, data, 0o644); err != nil {
		t.Fatalf("write legacy rules failed: %v", err)
	}

	app := &App{rulesPath: rulesPath}
	novels := app.defaultNovelsFromRules()
	if len(novels) != 1 {
		t.Fatalf("expected 1 migrated novel, got %d", len(novels))
	}

	novel := novels[0]
	if novel.RuleID != "zhswx" {
		t.Fatalf("expected migrated rule id zhswx, got %q", novel.RuleID)
	}
	if novel.Title != "青山" {
		t.Fatalf("expected migrated title 青山, got %q", novel.Title)
	}
	if novel.CatalogURL != "https://www.zhswx.com/chapter/67027.html" {
		t.Fatalf("expected migrated catalog url, got %q", novel.CatalogURL)
	}
}

func TestChapterCache_IsScopedPerRule(t *testing.T) {
	tempDir := t.TempDir()
	app := &App{configDir: tempDir}

	chapterURL := "https://example.com/book/1.html"
	if err := app.writeChapterCache("site-a", chapterURL, "chapter text"); err != nil {
		t.Fatalf("writeChapterCache returned error: %v", err)
	}

	text, found, err := app.readChapterCache("site-a", chapterURL)
	if err != nil {
		t.Fatalf("readChapterCache returned error: %v", err)
	}
	if !found || text != "chapter text" {
		t.Fatalf("expected cache hit for same rule, got found=%v text=%q", found, text)
	}

	text, found, err = app.readChapterCache("site-b", chapterURL)
	if err != nil {
		t.Fatalf("readChapterCache for other rule returned error: %v", err)
	}
	if found || text != "" {
		t.Fatalf("expected no cache hit across rules, got found=%v text=%q", found, text)
	}
}

func TestClearChapterCaches_RemovesOnlySelectedRules(t *testing.T) {
	tempDir := t.TempDir()
	app := &App{ctx: context.Background(), configDir: tempDir, rulesPath: filepath.Join(tempDir, "rules.json")}

	rules := []SiteRule{
		{ID: "site-a", Name: "Site A", MatchDomains: []string{"a.com"}},
		{ID: "site-b", Name: "Site B", MatchDomains: []string{"b.com"}},
	}
	if err := app.saveRules(rules); err != nil {
		t.Fatalf("saveRules returned error: %v", err)
	}
	if err := app.writeChapterCache("site-a", "https://a.com/1", "a"); err != nil {
		t.Fatalf("writeChapterCache site-a returned error: %v", err)
	}
	if err := app.writeChapterCache("site-b", "https://b.com/1", "b"); err != nil {
		t.Fatalf("writeChapterCache site-b returned error: %v", err)
	}

	entries, err := app.ClearChapterCaches([]string{"site-a"})
	if err != nil {
		t.Fatalf("ClearChapterCaches returned error: %v", err)
	}

	var siteA, siteB CacheEntry
	for _, entry := range entries {
		if entry.RuleID == "site-a" {
			siteA = entry
		}
		if entry.RuleID == "site-b" {
			siteB = entry
		}
	}
	if siteA.FileCount != 0 {
		t.Fatalf("expected site-a cache to be cleared, got %+v", siteA)
	}
	if siteB.FileCount != 1 {
		t.Fatalf("expected site-b cache to remain, got %+v", siteB)
	}
}

func TestFilterChaptersBySelection_KeepsCatalogOrder(t *testing.T) {
	chapters := []CatalogChapter{
		{Title: "1", URL: "u1"},
		{Title: "2", URL: "u2"},
		{Title: "3", URL: "u3"},
	}

	filtered, err := filterChaptersBySelection(chapters, []string{"u3", "u1"})
	if err != nil {
		t.Fatalf("filterChaptersBySelection returned error: %v", err)
	}
	if len(filtered) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(filtered))
	}
	if filtered[0].URL != "u1" || filtered[1].URL != "u3" {
		t.Fatalf("expected catalog order to be preserved, got %+v", filtered)
	}
}

func TestFilterChaptersBySelection_RejectsEmptyResult(t *testing.T) {
	_, err := filterChaptersBySelection([]CatalogChapter{{Title: "1", URL: "u1"}}, []string{"missing"})
	if err == nil {
		t.Fatal("expected error when no selected chapters match the catalog")
	}
}

func TestListNovelCaches_UsesIndexAndClearRemovesFiles(t *testing.T) {
	tempDir := t.TempDir()
	app := &App{configDir: tempDir}

	cachePath, ok := app.chapterCachePath("site-a", "https://a.com/1")
	if !ok {
		t.Fatal("expected cache path to resolve")
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		t.Fatalf("mkdir cache dir failed: %v", err)
	}
	if err := os.WriteFile(cachePath, []byte("chapter text"), 0o644); err != nil {
		t.Fatalf("write cache file failed: %v", err)
	}
	if err := app.upsertCacheRecord(ChapterCacheRecord{
		NovelID:      "novel-1",
		NovelTitle:   "Novel One",
		RuleID:       "site-a",
		RuleName:     "Site A",
		ChapterTitle: "Chapter 1",
		ChapterURL:   "https://a.com/1",
		CachePath:    cachePath,
	}); err != nil {
		t.Fatalf("upsertCacheRecord failed: %v", err)
	}

	entries, err := app.ListNovelCaches()
	if err != nil {
		t.Fatalf("ListNovelCaches failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 novel cache entry, got %d", len(entries))
	}
	if entries[0].NovelID != "novel-1" || entries[0].FileCount != 1 {
		t.Fatalf("unexpected novel cache entry: %+v", entries[0])
	}

	entries, err = app.ClearNovelCaches([]string{"novel-1"})
	if err != nil {
		t.Fatalf("ClearNovelCaches failed: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected caches to be empty after clear, got %+v", entries)
	}
	if _, err := os.Stat(cachePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected cache file to be removed, got err=%v", err)
	}
}

func TestSaveRule_ClearsCachesForUpdatedRule(t *testing.T) {
	tempDir := t.TempDir()
	app := &App{
		ctx:        context.Background(),
		configDir:  tempDir,
		rulesPath:  filepath.Join(tempDir, "rules.json"),
		novelsPath: filepath.Join(tempDir, "novels.json"),
	}

	originalRule := SiteRule{
		ID:                         "site-a",
		Name:                       "Site A",
		MatchDomains:               []string{"a.com"},
		CatalogChapterLinkSelector: ".catalog a",
		ChapterContentSelector:     "#content",
	}
	if err := app.saveRules([]SiteRule{originalRule}); err != nil {
		t.Fatalf("saveRules failed: %v", err)
	}
	if err := app.saveNovels([]Novel{{ID: "novel-1", Title: "Novel One", CatalogURL: "https://a.com/book", RuleID: "site-a"}}); err != nil {
		t.Fatalf("saveNovels failed: %v", err)
	}

	cachePath, ok := app.chapterCachePath("site-a", "https://a.com/book/1")
	if !ok {
		t.Fatal("expected cache path to resolve")
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		t.Fatalf("mkdir cache dir failed: %v", err)
	}
	if err := os.WriteFile(cachePath, []byte("cached chapter"), 0o644); err != nil {
		t.Fatalf("write cache file failed: %v", err)
	}
	if err := app.upsertCacheRecord(ChapterCacheRecord{
		NovelID:      "novel-1",
		NovelTitle:   "Novel One",
		RuleID:       "site-a",
		RuleName:     "Site A",
		ChapterTitle: "Chapter 1",
		ChapterURL:   "https://a.com/book/1",
		CachePath:    cachePath,
	}); err != nil {
		t.Fatalf("upsertCacheRecord failed: %v", err)
	}

	updatedRule := originalRule
	updatedRule.RemoveMatchingLines = []string{"提示语"}
	if _, err := app.SaveRule(updatedRule); err != nil {
		t.Fatalf("SaveRule failed: %v", err)
	}

	if _, err := os.Stat(cachePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected cache file to be removed after rule save, got err=%v", err)
	}

	entries, err := app.ListNovelCaches()
	if err != nil {
		t.Fatalf("ListNovelCaches failed: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected cache index to be cleared after rule save, got %+v", entries)
	}
}

func TestMarkCachedChapters_FlagsMatchingURLs(t *testing.T) {
	tempDir := t.TempDir()
	app := &App{configDir: tempDir}

	cachePath, ok := app.chapterCachePath("site-a", "https://a.com/2")
	if !ok {
		t.Fatal("expected cache path to resolve")
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		t.Fatalf("mkdir cache dir failed: %v", err)
	}
	if err := os.WriteFile(cachePath, []byte("chapter text"), 0o644); err != nil {
		t.Fatalf("write cache file failed: %v", err)
	}
	if err := app.upsertCacheRecord(ChapterCacheRecord{
		NovelID:      "novel-2",
		NovelTitle:   "Novel Two",
		RuleID:       "site-a",
		RuleName:     "Site A",
		ChapterTitle: "Chapter 2",
		ChapterURL:   "https://a.com/2",
		CachePath:    cachePath,
	}); err != nil {
		t.Fatalf("upsertCacheRecord failed: %v", err)
	}

	analysis := &CatalogAnalysis{
		Chapters: []CatalogChapter{
			{Title: "Chapter 1", URL: "https://a.com/1"},
			{Title: "Chapter 2", URL: "https://a.com/2"},
		},
	}
	if err := app.markCachedChapters(analysis, "novel-2", "site-a"); err != nil {
		t.Fatalf("markCachedChapters failed: %v", err)
	}
	if analysis.Chapters[0].Cached {
		t.Fatalf("expected first chapter to remain uncached: %+v", analysis.Chapters[0])
	}
	if !analysis.Chapters[1].Cached {
		t.Fatalf("expected second chapter to be marked cached: %+v", analysis.Chapters[1])
	}
}

func TestApplyTextCleanupRules_RemovesMatchingLinesAndReplacesText(t *testing.T) {
	input := "第一段\n请关闭浏览器阅读模式后查看本章节，否则将出现无法翻页或章节内容丢失等现象\n第二段 包含广告文案"
	output := applyTextCleanupRules(input, SiteRule{
		RemoveMatchingLines: []string{"请关闭浏览器阅读模式后查看本章节"},
		TextReplacementRules: []TextReplacementRule{
			{Match: "广告文案", Replace: "", Enabled: true},
		},
	})

	if strings.Contains(output, "阅读模式") {
		t.Fatalf("expected matching line to be removed, got %q", output)
	}
	if strings.Contains(output, "广告文案") {
		t.Fatalf("expected fixed text to be deleted, got %q", output)
	}
}

func TestApplyTextCleanupRules_SupportsRegexRules(t *testing.T) {
	input := "第1行\n作者提示：测试\n正文 ABC ABC"
	output := applyTextCleanupRules(input, SiteRule{
		RegexReplacementRules: []RegexReplacementRule{
			{Pattern: "^作者提示：.*$", RemoveLine: true, Enabled: true},
			{Pattern: "ABC", Replace: "XYZ", ReplaceFirst: true, Enabled: true},
		},
	})

	if strings.Contains(output, "作者提示") {
		t.Fatalf("expected regex line removal to run, got %q", output)
	}
	if strings.Count(output, "XYZ") != 1 || strings.Count(output, "ABC") != 1 {
		t.Fatalf("expected only first regex replacement, got %q", output)
	}
}
func TestReadChapter_UsesChapterCache(t *testing.T) {
	tempDir := t.TempDir()
	app := &App{ctx: context.Background(), configDir: tempDir}

	if err := app.writeChapterCache("site-a", "https://example.com/chapter-1", "cached text"); err != nil {
		t.Fatalf("writeChapterCache failed: %v", err)
	}

	result, err := app.ReadChapter(ChapterReadRequest{
		RuleID:       "site-a",
		ChapterURL:   "https://example.com/chapter-1",
		ChapterTitle: "Chapter 1",
	})
	if err != nil {
		t.Fatalf("ReadChapter failed: %v", err)
	}
	if !result.Cached || result.Content != "cached text" {
		t.Fatalf("unexpected read result: %+v", result)
	}
}

func TestReadChapter_RejectsEmptyChapterURL(t *testing.T) {
	app := &App{ctx: context.Background(), configDir: t.TempDir()}

	_, err := app.ReadChapter(ChapterReadRequest{RuleID: "site-a"})
	if err == nil {
		t.Fatal("expected error for empty chapter URL")
	}
}

func TestReadChapter_UsesNovelBindingAndCache(t *testing.T) {
	tempDir := t.TempDir()
	app := &App{
		ctx:        context.Background(),
		configDir:  tempDir,
		rulesPath:  filepath.Join(tempDir, "rules.json"),
		novelsPath: filepath.Join(tempDir, "novels.json"),
	}

	if err := app.saveRules([]SiteRule{{ID: "site-a", Name: "Site A"}}); err != nil {
		t.Fatalf("saveRules failed: %v", err)
	}
	if err := app.saveNovels([]Novel{{ID: "novel-1", Title: "Novel One", CatalogURL: "https://example.com/catalog", RuleID: "site-a"}}); err != nil {
		t.Fatalf("saveNovels failed: %v", err)
	}
	if err := app.writeChapterCache("site-a", "https://example.com/chapter-1", "cached text"); err != nil {
		t.Fatalf("writeChapterCache failed: %v", err)
	}

	result, err := app.ReadChapter(ChapterReadRequest{
		NovelID:      "novel-1",
		ChapterURL:   "https://example.com/chapter-1",
		ChapterTitle: "Chapter 1",
	})
	if err != nil {
		t.Fatalf("ReadChapter failed: %v", err)
	}
	if result.RuleID != "site-a" || result.NovelTitle != "Novel One" || !result.Cached || result.Content != "cached text" {
		t.Fatalf("unexpected read result: %+v", result)
	}
}
