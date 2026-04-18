# Keyword Highlight Filter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add keyword-based chapter highlighting and manual assist actions so users can visually spot chapters like 请假、感言、番外 without changing selection automatically.

**Architecture:** Extract keyword parsing and chapter matching into a small frontend utility module with node-native tests. Keep export selection untouched by default; only add highlight state, a matched-only view toggle, and current-page manual actions in the Vue screen.

**Tech Stack:** Vue 3, Vite, Node test runner

---

### Task 1: Chapter keyword utility helpers

**Files:**
- Create: `D:/go/wails/xiaoshuo/frontend/src/chapterFilters.js`
- Create: `D:/go/wails/xiaoshuo/frontend/src/chapterFilters.test.js`

- [ ] **Step 1: Write the failing test**

```js
import test from 'node:test'
import assert from 'node:assert/strict'
import { parseKeywordInput, matchChapterByKeywords } from './chapterFilters.js'

test('parseKeywordInput splits and trims comma-separated keywords', () => {
  assert.deepEqual(parseKeywordInput(' 请假, 感言 ,,番外 '), ['请假', '感言', '番外'])
})

test('matchChapterByKeywords matches title case-insensitively', () => {
  assert.equal(matchChapterByKeywords({ title: '第10章 番外篇' }, ['番外']), true)
  assert.equal(matchChapterByKeywords({ title: '正文章节' }, ['番外']), false)
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `node --test frontend/src/chapterFilters.test.js`
Expected: FAIL with missing module exports

- [ ] **Step 3: Write minimal implementation**

```js
export function parseKeywordInput(input) {
  return String(input || '')
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

export function matchChapterByKeywords(chapter, keywords) {
  const title = String(chapter?.title || '').toLowerCase()
  return keywords.some((keyword) => title.includes(String(keyword).toLowerCase()))
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `node --test frontend/src/chapterFilters.test.js`
Expected: PASS

### Task 2: Vue integration for highlight and manual assist tools

**Files:**
- Modify: `D:/go/wails/xiaoshuo/frontend/src/AppMain.vue`
- Modify: `D:/go/wails/xiaoshuo/frontend/src/style.css`

- [ ] **Step 1: Add highlight state and computed values**

Add export-form state for:
```js
keywordInput: '',
showMatchedOnly: false,
```

Import helpers and compute:
- parsed keywords
- matched chapter URL set
- paged chapter source that respects matched-only toggle

- [ ] **Step 2: Add toolbar controls**

Render controls near chapter selection toolbar:
```vue
<input v-model="exportForm.keywordInput" placeholder="例如：请假,感言,番外" />
<label><input v-model="exportForm.showMatchedOnly" type="checkbox" />仅查看匹配章节</label>
<button type="button" @click="selectMatchedOnCurrentPage">本页勾选匹配</button>
<button type="button" @click="clearMatchedOnCurrentPage">本页取消匹配</button>
```

- [ ] **Step 3: Add chapter highlight styling**

Matched chapters should visually stand out without changing selection automatically. Apply a conditional class such as `matched` to the chapter card and show a small badge like `匹配`.

- [ ] **Step 4: Build frontend**

Run: `npm run build`
Expected: PASS

### Task 3: Final verification

**Files:**
- Verify only

- [ ] **Step 1: Run frontend utility tests**

Run: `node --test frontend/src/chapterFilters.test.js`
Expected: PASS

- [ ] **Step 2: Regenerate bindings and build app**

Run: `wails generate module`
Expected: PASS

Run: `wails build`
Expected: PASS

- [ ] **Step 3: Manual behavior check**

Verify that:
- entering `请假,番外` highlights matching chapter titles
- toggling `仅查看匹配章节` filters preview only
- `本页勾选匹配 / 本页取消匹配` changes only current page selection
- export still uses the final checkbox selection, not the keyword input itself
