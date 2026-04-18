package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) ExportNovelAdvanced(req ExportRequest) (*ExportResult, error) {
	var novel *Novel
	if req.NovelID != "" {
		resolvedNovel, _, err := a.resolveNovel(req.NovelID)
		if err != nil {
			return nil, err
		}
		novel = resolvedNovel
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
	chapters, err = filterChaptersBySelection(chapters, req.SelectedChapterURLs)
	if err != nil {
		return nil, err
	}
	chapters, err = sliceChaptersByRange(chapters, req.StartChapter, req.EndChapter)
	if err != nil {
		return nil, err
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
		Filters: []runtime.FileFilter{{
			DisplayName: "Text Document (*.txt)",
			Pattern:     "*.txt",
		}},
	})
	if err != nil {
		return nil, err
	}
	if filePath == "" {
		return nil, errors.New("export cancelled")
	}

	a.emitProgress(ProgressEvent{Stage: "start", Message: "Starting extraction", Current: 0, Total: len(chapters)})

	failures := make([]ExportFailure, 0)
	tempDir, err := os.MkdirTemp("", "xiaoshuo-export-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	chapterFiles := make([]string, len(chapters))

	type chapterJob struct {
		index   int
		chapter CatalogChapter
	}
	type chapterResult struct {
		index       int
		chapter     CatalogChapter
		filePath    string
		retriesUsed int
		fromCache   bool
		err         error
	}

	workerCount := minInt(4, len(chapters))
	jobs := make(chan chapterJob)
	results := make(chan chapterResult, len(chapters))
	stop := make(chan struct{})

	var once sync.Once
	stopWorkers := func() {
		once.Do(func() {
			close(stop)
		})
	}

	var wg sync.WaitGroup
	for workerIndex := 0; workerIndex < workerCount; workerIndex++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				case job, ok := <-jobs:
					if !ok {
						return
					}

					text, retriesUsed, fromCache, err := a.extractChapterWithCache(rule, job.chapter, req.RetryCount)
					if err != nil {
						results <- chapterResult{
							index:       job.index,
							chapter:     job.chapter,
							retriesUsed: retriesUsed,
							err:         err,
						}
						continue
					}

					cachePath, ok := a.chapterCachePath(rule.ID, job.chapter.URL)
					if ok && novel != nil {
						if err := a.upsertCacheRecord(ChapterCacheRecord{
							NovelID:      novel.ID,
							NovelTitle:   novel.Title,
							RuleID:       rule.ID,
							RuleName:     rule.Name,
							ChapterTitle: job.chapter.Title,
							ChapterURL:   job.chapter.URL,
							CachePath:    cachePath,
							UpdatedAt:    time.Now().Format(time.RFC3339),
						}); err != nil {
							results <- chapterResult{
								index:       job.index,
								chapter:     job.chapter,
								retriesUsed: retriesUsed,
								err:         err,
							}
							continue
						}
					}

					chapterFile := filepath.Join(tempDir, fmt.Sprintf("%05d.txt", job.index+1))
					chapterBody := job.chapter.Title + "\n\n" + text + "\n\n"
					if err := os.WriteFile(chapterFile, []byte(chapterBody), 0o644); err != nil {
						results <- chapterResult{
							index:       job.index,
							chapter:     job.chapter,
							retriesUsed: retriesUsed,
							err:         err,
						}
						continue
					}

					results <- chapterResult{
						index:       job.index,
						chapter:     job.chapter,
						filePath:    chapterFile,
						retriesUsed: retriesUsed,
						fromCache:   fromCache,
					}
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for index, chapter := range chapters {
			select {
			case <-stop:
				return
			case jobs <- chapterJob{index: index, chapter: chapter}:
			}
		}
	}()

	completed := 0
	for completed < len(chapters) {
		result := <-results
		completed++

		if result.err != nil {
			if !req.SkipOnFailure {
				stopWorkers()
				wg.Wait()
				return nil, fmt.Errorf("failed to fetch chapter %s: %w", result.chapter.Title, result.err)
			}

			failures = append(failures, ExportFailure{
				Index:   result.index + 1,
				Title:   result.chapter.Title,
				URL:     result.chapter.URL,
				Error:   result.err.Error(),
				Retries: result.retriesUsed,
			})

			a.emitProgress(ProgressEvent{
				Stage:        "warning",
				Message:      fmt.Sprintf("Skipped chapter %d/%d after failure", result.index+1, len(chapters)),
				Current:      completed,
				Total:        len(chapters),
				ChapterTitle: result.chapter.Title,
			})
			continue
		}

		chapterFiles[result.index] = result.filePath
		message := fmt.Sprintf("Fetched chapter %d/%d", result.index+1, len(chapters))
		if result.fromCache {
			message = fmt.Sprintf("Loaded chapter %d/%d from cache", result.index+1, len(chapters))
		}
		a.emitProgress(ProgressEvent{
			Stage:        "chapter",
			Message:      message,
			Current:      completed,
			Total:        len(chapters),
			ChapterTitle: result.chapter.Title,
		})
	}

	stopWorkers()
	wg.Wait()

	output, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer output.Close()

	if _, err := output.WriteString(novelTitle + "\n\n"); err != nil {
		return nil, err
	}

	for _, chapterFile := range chapterFiles {
		if chapterFile == "" {
			continue
		}
		file, err := os.Open(chapterFile)
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(output, file); err != nil {
			file.Close()
			return nil, err
		}
		file.Close()
	}

	if len(failures) > 0 {
		if _, err := output.WriteString("Failed Chapters\n\n"); err != nil {
			return nil, err
		}
		for _, failure := range failures {
			if _, err := output.WriteString(fmt.Sprintf("%d. %s\n", failure.Index, failure.Title)); err != nil {
				return nil, err
			}
			if _, err := output.WriteString(fmt.Sprintf("URL: %s\n", failure.URL)); err != nil {
				return nil, err
			}
			if _, err := output.WriteString(fmt.Sprintf("Error: %s\n\n", failure.Error)); err != nil {
				return nil, err
			}
		}
	}

	a.emitProgress(ProgressEvent{Stage: "done", Message: "Export finished", Current: len(chapters), Total: len(chapters)})

	return &ExportResult{
		FilePath:      filePath,
		RuleID:        rule.ID,
		NovelTitle:    novelTitle,
		ExportedCount: len(chapters) - len(failures),
		FailureCount:  len(failures),
		Failures:      failures,
	}, nil
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func sliceChaptersByRange(chapters []CatalogChapter, startChapter, endChapter int) ([]CatalogChapter, error) {
	if startChapter < 0 || endChapter < 0 {
		return nil, errors.New("chapter range cannot be negative")
	}
	if startChapter == 0 {
		startChapter = 1
	}
	if endChapter == 0 {
		endChapter = len(chapters)
	}
	if startChapter > endChapter {
		return nil, errors.New("start chapter cannot be greater than end chapter")
	}
	if startChapter > len(chapters) {
		return nil, errors.New("start chapter is out of range")
	}
	if endChapter > len(chapters) {
		endChapter = len(chapters)
	}
	return chapters[startChapter-1 : endChapter], nil
}
