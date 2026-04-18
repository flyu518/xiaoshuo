package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

func (a *App) SaveNovel(novel Novel) (*AppState, error) {
	rules, err := a.loadRules()
	if err != nil {
		return nil, err
	}
	novels, err := a.loadNovels()
	if err != nil {
		return nil, err
	}

	novel = normalizeNovel(novel)
	if novel.ID == "" {
		novel.ID = fmt.Sprintf("novel-%d", time.Now().UnixNano())
	}

	if novel.RuleID == "" {
		return nil, errors.New("novel must be bound to a site rule")
	}
	rule, err := a.resolveRule(novel.RuleID, novel.CatalogURL)
	if err != nil {
		return nil, err
	}
	if !ruleMatchesURL(*rule, novel.CatalogURL) {
		return nil, errors.New("当前目录地址与所选规则不匹配，请切换规则或更换目录地址")
	}

	index := slices.IndexFunc(novels, func(item Novel) bool {
		return item.ID == novel.ID
	})
	if index >= 0 {
		novels[index] = novel
	} else {
		novels = append(novels, novel)
	}

	if err := a.saveNovels(novels); err != nil {
		return nil, err
	}
	return &AppState{Rules: rules, Novels: novels}, nil
}

func (a *App) DeleteNovel(novelID string) (*AppState, error) {
	rules, err := a.loadRules()
	if err != nil {
		return nil, err
	}
	novels, err := a.loadNovels()
	if err != nil {
		return nil, err
	}

	filtered := make([]Novel, 0, len(novels))
	for _, novel := range novels {
		if novel.ID != novelID {
			filtered = append(filtered, novel)
		}
	}
	if len(filtered) == len(novels) {
		return nil, errors.New("novel not found")
	}
	if err := a.saveNovels(filtered); err != nil {
		return nil, err
	}

	return &AppState{Rules: rules, Novels: filtered}, nil
}

func (a *App) ensureNovelsFile() error {
	if _, err := os.Stat(a.novelsPath); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	seed := a.defaultNovelsFromRules()
	return a.saveNovels(seed)
}

func (a *App) loadNovels() ([]Novel, error) {
	if err := a.ensureNovelsFile(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(a.novelsPath)
	if err != nil {
		return nil, err
	}

	var novels []Novel
	if len(data) > 0 {
		if err := json.Unmarshal(data, &novels); err != nil {
			return nil, err
		}
	}
	for index, novel := range novels {
		novels[index] = normalizeNovel(novel)
	}
	return novels, nil
}

func (a *App) saveNovels(novels []Novel) error {
	for index, novel := range novels {
		novels[index] = normalizeNovel(novel)
	}

	data, err := json.MarshalIndent(novels, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.novelsPath, data, 0o644)
}

func (a *App) resolveNovel(novelID string) (*Novel, *SiteRule, error) {
	novels, err := a.loadNovels()
	if err != nil {
		return nil, nil, err
	}
	for _, novel := range novels {
		if novel.ID == novelID {
			rule, err := a.resolveRule(novel.RuleID, novel.CatalogURL)
			if err != nil {
				return nil, nil, err
			}
			copiedNovel := novel
			return &copiedNovel, rule, nil
		}
	}
	return nil, nil, errors.New("specified novel was not found")
}

func (a *App) defaultNovelsFromRules() []Novel {
	data, err := os.ReadFile(a.rulesPath)
	if err != nil || len(data) == 0 {
		return []Novel{}
	}

	type legacyRule struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		CatalogURL string `json:"catalogUrl"`
		NovelTitle string `json:"novelTitle"`
	}

	var rules []legacyRule
	if err := json.Unmarshal(data, &rules); err != nil {
		return []Novel{}
	}

	result := make([]Novel, 0, len(rules))
	for _, rule := range rules {
		catalogURL := strings.TrimSpace(rule.CatalogURL)
		if catalogURL == "" {
			continue
		}

		title := strings.TrimSpace(rule.NovelTitle)
		if title == "" {
			title = strings.TrimSpace(rule.Name)
		}
		if title == "" {
			title = inferNovelTitleFromCatalogURL(catalogURL)
		}

		ruleID := strings.TrimSpace(rule.ID)
		if ruleID == "" {
			continue
		}

		result = append(result, Novel{
			ID:         ruleID + "-default",
			Title:      title,
			CatalogURL: catalogURL,
			RuleID:     ruleID,
		})
	}
	return result
}

func inferNovelTitleFromCatalogURL(rawURL string) string {
	base := filepath.Base(strings.TrimRight(rawURL, "/"))
	base = strings.TrimSuffix(base, filepath.Ext(base))
	base = strings.TrimSpace(base)
	if base == "" || base == "." {
		return ""
	}
	return base
}

func normalizeNovel(novel Novel) Novel {
	novel.ID = strings.TrimSpace(novel.ID)
	novel.Title = strings.TrimSpace(novel.Title)
	novel.CatalogURL = strings.TrimSpace(novel.CatalogURL)
	novel.RuleID = strings.TrimSpace(novel.RuleID)
	return novel
}
