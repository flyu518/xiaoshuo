package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type CacheEntry struct {
	RuleID     string `json:"ruleId"`
	RuleName   string `json:"ruleName"`
	FileCount  int    `json:"fileCount"`
	TotalBytes int64  `json:"totalBytes"`
}

func (a *App) readChapterCache(ruleID, chapterURL string) (string, bool, error) {
	cachePath, ok := a.chapterCachePath(ruleID, chapterURL)
	if !ok {
		return "", false, nil
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}
	return string(data), true, nil
}

func (a *App) writeChapterCache(ruleID, chapterURL, text string) error {
	cachePath, ok := a.chapterCachePath(ruleID, chapterURL)
	if !ok {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(cachePath, []byte(text), 0o644)
}

func (a *App) extractChapterWithCache(rule *SiteRule, chapter CatalogChapter, retryCount int) (string, int, bool, error) {
	if rule != nil && rule.ID != "" {
		text, found, err := a.readChapterCache(rule.ID, chapter.URL)
		if err != nil {
			return "", 0, false, err
		}
		if found && strings.TrimSpace(text) != "" {
			return text, 0, true, nil
		}
	}

	text, retriesUsed, err := a.extractChapterWithRetry(rule, chapter, retryCount)
	if err != nil {
		return "", retriesUsed, false, err
	}

	if rule != nil && rule.ID != "" && strings.TrimSpace(text) != "" {
		if err := a.writeChapterCache(rule.ID, chapter.URL, text); err != nil {
			return "", retriesUsed, false, err
		}
	}

	return text, retriesUsed, false, nil
}

func (a *App) chapterCachePath(ruleID, chapterURL string) (string, bool) {
	if strings.TrimSpace(a.configDir) == "" || strings.TrimSpace(ruleID) == "" || strings.TrimSpace(chapterURL) == "" {
		return "", false
	}

	hash := sha256.Sum256([]byte(strings.TrimSpace(chapterURL)))
	fileName := hex.EncodeToString(hash[:]) + ".txt"
	return filepath.Join(a.chapterCacheRoot(), sanitizeFilename(ruleID), fileName), true
}

func (a *App) ListChapterCaches() ([]CacheEntry, error) {
	rules, err := a.loadRules()
	if err != nil {
		return nil, err
	}

	entries := make([]CacheEntry, 0, len(rules))
	root := a.chapterCacheRoot()
	for _, rule := range rules {
		entry := CacheEntry{
			RuleID:   rule.ID,
			RuleName: rule.Name,
		}

		ruleDir := filepath.Join(root, sanitizeFilename(rule.ID))
		files, err := os.ReadDir(ruleDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				entries = append(entries, entry)
				continue
			}
			return nil, err
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}
			info, err := file.Info()
			if err != nil {
				return nil, err
			}
			entry.FileCount++
			entry.TotalBytes += info.Size()
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (a *App) ClearChapterCaches(ruleIDs []string) ([]CacheEntry, error) {
	root := a.chapterCacheRoot()
	for _, ruleID := range ruleIDs {
		ruleID = strings.TrimSpace(ruleID)
		if ruleID == "" {
			continue
		}
		ruleDir := filepath.Join(root, sanitizeFilename(ruleID))
		if err := os.RemoveAll(ruleDir); err != nil {
			return nil, err
		}
	}

	return a.ListChapterCaches()
}

func (a *App) chapterCacheRoot() string {
	return filepath.Join(a.configDir, "chapter-cache")
}
