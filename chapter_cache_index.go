package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type ChapterCacheRecord struct {
	NovelID      string `json:"novelId"`
	NovelTitle   string `json:"novelTitle"`
	RuleID       string `json:"ruleId"`
	RuleName     string `json:"ruleName"`
	ChapterTitle string `json:"chapterTitle"`
	ChapterURL   string `json:"chapterUrl"`
	CachePath    string `json:"cachePath"`
	UpdatedAt    string `json:"updatedAt"`
}

type NovelCacheEntry struct {
	NovelID      string   `json:"novelId"`
	NovelTitle   string   `json:"novelTitle"`
	RuleID       string   `json:"ruleId"`
	RuleName     string   `json:"ruleName"`
	FileCount    int      `json:"fileCount"`
	TotalBytes   int64    `json:"totalBytes"`
	CachedTitles []string `json:"cachedTitles,omitempty"`
}

func (a *App) loadCacheIndex() ([]ChapterCacheRecord, error) {
	a.cacheMu.Lock()
	defer a.cacheMu.Unlock()
	return a.loadCacheIndexUnlocked()
}

func (a *App) loadCacheIndexUnlocked() ([]ChapterCacheRecord, error) {
	path := a.cacheIndexPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []ChapterCacheRecord{}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return []ChapterCacheRecord{}, nil
	}

	var records []ChapterCacheRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}
	for index, record := range records {
		records[index] = normalizeCacheRecord(record)
	}
	return records, nil
}

func (a *App) saveCacheIndex(records []ChapterCacheRecord) error {
	a.cacheMu.Lock()
	defer a.cacheMu.Unlock()
	return a.saveCacheIndexUnlocked(records)
}

func (a *App) saveCacheIndexUnlocked(records []ChapterCacheRecord) error {
	for index, record := range records {
		records[index] = normalizeCacheRecord(record)
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.cacheIndexPath(), data, 0o644)
}

func (a *App) upsertCacheRecord(record ChapterCacheRecord) error {
	record = normalizeCacheRecord(record)
	if record.ChapterURL == "" || record.CachePath == "" || record.RuleID == "" {
		return nil
	}

	a.cacheMu.Lock()
	defer a.cacheMu.Unlock()

	records, err := a.loadCacheIndexUnlocked()
	if err != nil {
		return err
	}
	matchIndex := slices.IndexFunc(records, func(item ChapterCacheRecord) bool {
		return item.RuleID == record.RuleID && item.NovelID == record.NovelID && item.ChapterURL == record.ChapterURL
	})
	if matchIndex >= 0 {
		records[matchIndex] = record
	} else {
		records = append(records, record)
	}
	return a.saveCacheIndexUnlocked(records)
}

func (a *App) cachedChapterURLsForNovel(novelID, ruleID string) (map[string]struct{}, error) {
	records, err := a.loadCacheIndex()
	if err != nil {
		return nil, err
	}

	urls := map[string]struct{}{}
	for _, record := range records {
		if strings.TrimSpace(record.NovelID) != strings.TrimSpace(novelID) {
			continue
		}
		if strings.TrimSpace(ruleID) != "" && strings.TrimSpace(record.RuleID) != strings.TrimSpace(ruleID) {
			continue
		}
		if record.ChapterURL != "" {
			urls[record.ChapterURL] = struct{}{}
		}
	}
	return urls, nil
}

func (a *App) ListNovelCaches() ([]NovelCacheEntry, error) {
	a.cacheMu.Lock()
	defer a.cacheMu.Unlock()

	records, err := a.loadCacheIndexUnlocked()
	if err != nil {
		return nil, err
	}

	type aggregate struct {
		entry    NovelCacheEntry
		titleSet map[string]struct{}
	}

	entriesByNovel := map[string]*aggregate{}
	cleaned := make([]ChapterCacheRecord, 0, len(records))
	for _, record := range records {
		record = normalizeCacheRecord(record)
		if record.NovelID == "" || record.CachePath == "" {
			continue
		}

		info, err := os.Stat(record.CachePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}

		key := record.RuleID + "::" + record.NovelID
		agg, ok := entriesByNovel[key]
		if !ok {
			agg = &aggregate{
				entry: NovelCacheEntry{
					NovelID:    record.NovelID,
					NovelTitle: record.NovelTitle,
					RuleID:     record.RuleID,
					RuleName:   record.RuleName,
				},
				titleSet: map[string]struct{}{},
			}
			entriesByNovel[key] = agg
		}

		agg.entry.FileCount++
		agg.entry.TotalBytes += info.Size()
		if record.ChapterTitle != "" {
			agg.titleSet[record.ChapterTitle] = struct{}{}
		}
		cleaned = append(cleaned, record)
	}

	if len(cleaned) != len(records) {
		if err := a.saveCacheIndexUnlocked(cleaned); err != nil {
			return nil, err
		}
	}

	entries := make([]NovelCacheEntry, 0, len(entriesByNovel))
	for _, agg := range entriesByNovel {
		for title := range agg.titleSet {
			agg.entry.CachedTitles = append(agg.entry.CachedTitles, title)
		}
		slices.Sort(agg.entry.CachedTitles)
		entries = append(entries, agg.entry)
	}

	slices.SortFunc(entries, func(left, right NovelCacheEntry) int {
		if left.RuleName != right.RuleName {
			return strings.Compare(left.RuleName, right.RuleName)
		}
		return strings.Compare(left.NovelTitle, right.NovelTitle)
	})
	return entries, nil
}

func (a *App) ClearNovelCaches(novelIDs []string) ([]NovelCacheEntry, error) {
	selected := map[string]struct{}{}
	for _, novelID := range novelIDs {
		novelID = strings.TrimSpace(novelID)
		if novelID != "" {
			selected[novelID] = struct{}{}
		}
	}

	a.cacheMu.Lock()

	records, err := a.loadCacheIndexUnlocked()
	if err != nil {
		a.cacheMu.Unlock()
		return nil, err
	}

	kept := make([]ChapterCacheRecord, 0, len(records))
	for _, record := range records {
		if _, ok := selected[record.NovelID]; ok {
			if record.CachePath != "" {
				_ = os.Remove(record.CachePath)
			}
			continue
		}
		kept = append(kept, record)
	}

	if err := a.saveCacheIndexUnlocked(kept); err != nil {
		a.cacheMu.Unlock()
		return nil, err
	}
	a.cacheMu.Unlock()
	return a.ListNovelCaches()
}

func (a *App) clearRuleCaches(ruleID string) error {
	ruleID = strings.TrimSpace(ruleID)
	if ruleID == "" {
		return nil
	}

	a.cacheMu.Lock()
	defer a.cacheMu.Unlock()

	records, err := a.loadCacheIndexUnlocked()
	if err != nil {
		return err
	}

	kept := make([]ChapterCacheRecord, 0, len(records))
	for _, record := range records {
		if strings.TrimSpace(record.RuleID) == ruleID {
			if record.CachePath != "" {
				_ = os.Remove(record.CachePath)
			}
			continue
		}
		kept = append(kept, record)
	}

	if err := os.RemoveAll(filepath.Join(a.chapterCacheRoot(), sanitizeFilename(ruleID))); err != nil {
		return err
	}

	return a.saveCacheIndexUnlocked(kept)
}

func normalizeCacheRecord(record ChapterCacheRecord) ChapterCacheRecord {
	record.NovelID = strings.TrimSpace(record.NovelID)
	record.NovelTitle = strings.TrimSpace(record.NovelTitle)
	record.RuleID = strings.TrimSpace(record.RuleID)
	record.RuleName = strings.TrimSpace(record.RuleName)
	record.ChapterTitle = strings.TrimSpace(record.ChapterTitle)
	record.ChapterURL = strings.TrimSpace(record.ChapterURL)
	record.CachePath = strings.TrimSpace(record.CachePath)
	record.UpdatedAt = strings.TrimSpace(record.UpdatedAt)
	if record.UpdatedAt == "" {
		record.UpdatedAt = time.Now().Format(time.RFC3339)
	}
	return record
}

func (a *App) cacheIndexPath() string {
	return filepath.Join(a.configDir, "chapter-cache-index.json")
}
