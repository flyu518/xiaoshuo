# Chapter Selection Export Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add chapter-range selection tools for very large catalogs so users can choose export chapters by range input plus paged checkboxes instead of start/end chapter fields.

**Architecture:** Keep catalog analysis as the source of truth, then maintain a frontend selection set keyed by chapter URL. Extend the export request with selected chapter URLs and filter analyzed chapters on the backend before the existing cache/concurrency export pipeline runs.

**Tech Stack:** Go, Wails, Vue 3, Vite

---

### Task 1: Backend selected-chapter filtering

**Files:**
- Modify: `D:/go/wails/xiaoshuo/app.go`
- Modify: `D:/go/wails/xiaoshuo/export_advanced.go`
- Test: `D:/go/wails/xiaoshuo/app_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestFilterChaptersBySelection_KeepsSelectedOrder(t *testing.T) {
    chapters := []CatalogChapter{{Title: "1", URL: "u1"}, {Title: "2", URL: "u2"}, {Title: "3", URL: "u3"}}
    filtered, err := filterChaptersBySelection(chapters, []string{"u3", "u1"})
    if err != nil {
        t.Fatalf("filterChaptersBySelection returned error: %v", err)
    }
    if len(filtered) != 2 || filtered[0].URL != "u1" || filtered[1].URL != "u3" {
        t.Fatalf("unexpected filtered chapters: %+v", filtered)
    }
}

func TestFilterChaptersBySelection_RejectsEmptySelection(t *testing.T) {
    _, err := filterChaptersBySelection([]CatalogChapter{{Title: "1", URL: "u1"}}, []string{"missing"})
    if err == nil {
        t.Fatal("expected error for empty filtered result")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./...`
Expected: FAIL with `undefined: filterChaptersBySelection`

- [ ] **Step 3: Write minimal implementation**

```go
type ExportRequest struct {
    // existing fields...
    SelectedChapterURLs []string `json:"selectedChapterUrls"`
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
```

Then call `filterChaptersBySelection()` near the top of `ExportNovelAdvanced` before range/limit logic.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add app.go export_advanced.go app_test.go
git commit -m "feat: support selected chapter export"
```

### Task 2: Frontend selection model and batch tools

**Files:**
- Modify: `D:/go/wails/xiaoshuo/frontend/src/AppMain.vue`

- [ ] **Step 1: Add selection state and helper functions**

Add reactive state for:
```js
const exportForm = reactive({
  novelId: '',
  maxChapters: 0,
  retryCount: 2,
  skipOnFailure: true,
  skipFilteredTitle: true,
  selectedChapterUrls: [],
  rangeInput: '',
  currentPage: 1,
  pageSize: 100,
})
```

Add helpers to:
- reset selection after fresh analysis
- select all / clear / invert
- parse `1-200,250,300-320`
- toggle current page chapter URLs
- compute paged chapters and selected count

- [ ] **Step 2: Replace start/end controls with selection controls**

Render a new chapter selection toolbar above preview:
```vue
<div class="selection-tools">
  <input v-model="exportForm.rangeInput" placeholder="例如：1-200,520-560" />
  <button type="button" @click="applyRangeSelection">应用范围</button>
  <button type="button" @click="selectAllChapters">全选</button>
  <button type="button" @click="clearChapterSelection">清空</button>
  <button type="button" @click="invertChapterSelection">反选</button>
</div>
```

In the preview list, add checkboxes for the current page only and paging controls for previous/next page.

- [ ] **Step 3: Wire export to selected URLs only**

Send:
```js
selectedChapterUrls: [...exportForm.selectedChapterUrls],
```

Block export with a user-facing error if no chapters are selected.

- [ ] **Step 4: Build frontend**

Run: `npm run build`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/AppMain.vue
git commit -m "feat: add chapter selection controls"
```

### Task 3: End-to-end verification

**Files:**
- Verify only

- [ ] **Step 1: Regenerate bindings**

Run: `wails generate module`
Expected: PASS

- [ ] **Step 2: Build application**

Run: `wails build`
Expected: PASS and output binary at `build/bin/xiaoshuo.exe`

- [ ] **Step 3: Manual behavior check**

Verify in `wails dev` or built app:
- analyzing a novel resets selection to all chapters selected
- applying `1-10,20-25` selects only those chapters
- paging preserves selection state
- export refuses to run with zero chapters selected

- [ ] **Step 4: Commit**

```bash
git add docs/superpowers/plans/2026-04-17-chapter-selection-export.md
git commit -m "docs: add chapter selection export plan"
```
