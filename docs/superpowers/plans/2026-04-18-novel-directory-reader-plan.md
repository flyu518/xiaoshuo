# Novel Directory Reader Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a novel browsing flow where the user can open a directory from the novel list, click any chapter, and read that chapter in a dedicated reading view while reusing the same chapter data used for export.

**Architecture:** Keep the existing top-level tabs, but turn the Novel tab into a small workspace with three responsibilities: novel management, chapter directory browsing, and chapter reading. Add a backend chapter-read endpoint that reuses the existing chapter extraction and cache logic, then let the frontend handle directory navigation and chapter-to-chapter movement from the analyzed chapter list.

**Tech Stack:** Go, Wails, Vue 3, CSS Grid/Flexbox, Node's built-in `assert`

---

### Task 1: Add a backend single-chapter reader API

**Files:**
- Modify: `app.go`
- Modify: `app_test.go`

- [ ] **Step 1: Add the request/response types and backend method**

Define a new request and response pair near the other app API types:

```go
type ChapterReadRequest struct {
	CatalogURL  string `json:"catalogUrl"`
	RuleID      string `json:"ruleId"`
	NovelID     string `json:"novelId"`
	ChapterURL  string `json:"chapterUrl"`
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
```

Implement `func (a *App) ReadChapter(req ChapterReadRequest) (*ChapterReadResult, error)` so it:
- resolves the novel and rule when `NovelID` is present
- resolves the rule directly when `RuleID`/`CatalogURL` are provided
- returns a clear error when `ChapterURL` is empty
- checks the chapter cache first
- falls back to the existing `extractChapter(rule, CatalogChapter{Title: ..., URL: ...})`
- writes fresh chapter text back to cache when it had to fetch

- [ ] **Step 2: Add failing tests for cache-first reading**

Add tests that prove the new API prefers cache and rejects empty input:

```go
func TestReadChapter_UsesChapterCache(t *testing.T) {
	tempDir := t.TempDir()
	app := &App{ctx: context.Background(), configDir: tempDir}

	if err := app.writeChapterCache("site-a", "https://example.com/chapter-1", "cached text"); err != nil {
		t.Fatalf("writeChapterCache failed: %v", err)
	}

	result, err := app.ReadChapter(ChapterReadRequest{
		RuleID:      "site-a",
		ChapterURL:  "https://example.com/chapter-1",
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
```

- [ ] **Step 3: Implement the minimal backend logic to make the tests pass**

Keep the change small: reuse `resolveNovel`, `resolveRule`, `readChapterCache`, `writeChapterCache`, and `extractChapter` instead of creating a parallel fetch stack.

- [ ] **Step 4: Run the Go test suite**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 5: Commit the backend step**

```bash
git add app.go app_test.go
git commit -m "feat: add single chapter reader backend"
```

### Task 2: Add frontend reader state and chapter navigation helpers

**Files:**
- Create: `frontend/src/novelReader.js`
- Create: `frontend/src/novelReader.test.js`
- Modify: `frontend/src/AppMain.vue`

- [ ] **Step 1: Add a small helper module for chapter navigation**

Create helper functions that keep the chapter reader logic out of the giant Vue file:

```js
export function findChapterIndex(chapters, chapterUrl) {
  return chapters.findIndex((chapter) => chapter.url === chapterUrl)
}

export function getChapterNeighbors(chapters, chapterUrl) {
  const index = findChapterIndex(chapters, chapterUrl)
  return {
    index,
    previous: index > 0 ? chapters[index - 1] : null,
    current: index >= 0 ? chapters[index] : null,
    next: index >= 0 && index < chapters.length - 1 ? chapters[index + 1] : null,
  }
}

export function normalizeNovelWorkspaceView(view) {
  return view === 'directory' || view === 'reading' ? view : 'directory'
}
```

- [ ] **Step 2: Add a failing helper test**

Use the same style as the existing helper tests in `frontend/src/*.test.js`:

```js
import assert from 'node:assert/strict'
import { findChapterIndex, getChapterNeighbors, normalizeNovelWorkspaceView } from './novelReader.js'

const chapters = [
  { title: 'Chapter 1', url: 'u1' },
  { title: 'Chapter 2', url: 'u2' },
  { title: 'Chapter 3', url: 'u3' },
]

assert.equal(findChapterIndex(chapters, 'u2'), 1)
assert.deepEqual(getChapterNeighbors(chapters, 'u2'), {
  index: 1,
  previous: chapters[0],
  current: chapters[1],
  next: chapters[2],
})
assert.equal(normalizeNovelWorkspaceView('bad-value'), 'directory')
```

- [ ] **Step 3: Add novel-reader UI state and actions to `AppMain.vue`**

Introduce a small novel-workspace state object in the Vue component to track:
- active novel id
- current workspace view (`directory` or `reading`)
- active chapter url
- loaded chapter content
- loading and error state for chapter reading
- directory paging and directory search

Wire the Novel tab so it can:
- open the directory for the selected novel
- load a chapter into the reading panel
- move to previous/next chapter using the shared chapter list
- return from reading to directory without clearing the selected novel

- [ ] **Step 4: Run the helper test and the frontend build**

Run:
- `node frontend/src/novelReader.test.js`
- `npm run build`

Expected:
- helper test exits cleanly with no assertion failure
- frontend build succeeds

- [ ] **Step 5: Commit the frontend reader-state step**

```bash
git add frontend/src/AppMain.vue frontend/src/novelReader.js frontend/src/novelReader.test.js
git commit -m "feat: add novel directory and reader state"
```

### Task 3: Split the Novel tab into directory and reading panels

**Files:**
- Modify: `frontend/src/AppMain.vue`
- Modify: `frontend/src/style.css`

- [ ] **Step 1: Restructure the Novel tab layout**

Update the existing Novel tab so the selected novel can open a directory panel and a reading panel side by side on desktop. Keep the novel list and novel editor available, but make the directory and reading area the main focus once a novel is opened.

- [ ] **Step 2: Add chapter click behavior in the directory panel**

Make each chapter row selectable so clicking it opens the chapter in the reading panel. Show cached chapter badges in the directory, but keep the reading panel focused on the current chapter text and navigation buttons.

- [ ] **Step 3: Add responsive layout styles**

Add styles for:
- a novel workspace grid
- a directory panel
- a reading panel
- compact chapter navigation controls
- a mobile breakpoint where the directory and reading panels stack vertically

Keep the reading panel scrollable so long chapter text does not grow the whole page.

- [ ] **Step 4: Run the frontend build again**

Run: `npm run build`
Expected: PASS

- [ ] **Step 5: Commit the layout step**

```bash
git add frontend/src/AppMain.vue frontend/src/style.css
git commit -m "feat: split novel workspace into directory and reading panels"
```

### Task 4: Verify the end-to-end flow

**Files:**
- Modify: `app_test.go`
- Modify: `frontend/src/novelReader.test.js`
- Test: `go test ./...`
- Test: `npm run build`
- Test: `wails build`

- [ ] **Step 1: Expand Go coverage for the new read path**

Add one more backend test that confirms the reader API uses the resolved novel rule when `NovelID` is supplied and still returns cached content for a matching chapter URL.

- [ ] **Step 2: Run the full Go test suite**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 3: Run the frontend and desktop builds**

Run:
- `npm run build`
- `wails build`

Expected:
- both builds succeed with no layout or runtime errors

- [ ] **Step 4: Commit the verification step**

```bash
git add app_test.go frontend/src/novelReader.test.js
git commit -m "test: verify novel reader flow"
```

## Coverage Check

- Novel list and novel selection: Task 2 and Task 3
- Directory browsing and chapter click-through: Task 2 and Task 3
- Chapter reading and previous/next navigation: Task 1 and Task 2
- Cache-first chapter loading: Task 1 and Task 4
- Responsive desktop/mobile layout: Task 3
- Export behavior staying unchanged: Task 2 and Task 4

## Gaps

No known gaps remain. The plan adds the backend API needed for reading, keeps chapter data shared with export, and gives the Novel tab a dedicated directory/reading flow without expanding the top-level tab set.
