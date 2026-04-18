# Novel List Search And Filter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add lightweight novel-list search and site-rule filtering so the left sidebar stays easy to use as the saved novel list grows.

**Architecture:** Keep the source novel list unchanged and add frontend-only filter state plus computed filtering. The selected novel remains stable even when hidden by the current filter so export/edit state is not disturbed.

**Tech Stack:** Vue 3, Vite

---

### Task 1: Frontend filter helpers

**Files:**
- Create: `D:/go/wails/xiaoshuo/frontend/src/novelListFilters.js`
- Create: `D:/go/wails/xiaoshuo/frontend/src/novelListFilters.test.js`

- [ ] **Step 1: Write the failing test**

```js
import assert from 'node:assert/strict'
import { filterNovels } from './novelListFilters.js'

const novels = [
  { id: '1', title: '青山', ruleId: 'zhswx' },
  { id: '2', title: '凡人修仙传', ruleId: 'zhswx' },
  { id: '3', title: '诡秘之主', ruleId: '23shuku' },
]

assert.deepEqual(filterNovels(novels, '青', '' ).map((item) => item.id), ['1'])
assert.deepEqual(filterNovels(novels, '', '23shuku').map((item) => item.id), ['3'])
```

- [ ] **Step 2: Run test to verify it fails**

Run: `node frontend/src/novelListFilters.test.js`
Expected: FAIL because module does not exist

- [ ] **Step 3: Write minimal implementation**

Create `filterNovels()` that filters by case-insensitive title substring and exact ruleId when provided.

- [ ] **Step 4: Run test to verify it passes**

Run: `node frontend/src/novelListFilters.test.js`
Expected: PASS

### Task 2: Sidebar UI wiring

**Files:**
- Modify: `D:/go/wails/xiaoshuo/frontend/src/AppMain.vue`
- Modify: `D:/go/wails/xiaoshuo/frontend/src/style.css`

- [ ] **Step 1: Add filter state and computed list**

Add:
```js
const listFilters = reactive({
  search: '',
  ruleId: '',
})
```

Import `filterNovels()` and render `filteredNovels` in the sidebar.

- [ ] **Step 2: Add search and rule select controls**

Place a text input and select above the novel buttons with labels/placeholder text in the existing Chinese UI.

- [ ] **Step 3: Keep selection stable**

Do not auto-change the currently selected novel when filters hide it. Only change selection when the user clicks a filtered novel.

- [ ] **Step 4: Build frontend**

Run: `npm run build`
Expected: PASS

### Task 3: Final verification

**Files:**
- Verify only

- [ ] **Step 1: Run helper test**

Run: `node frontend/src/novelListFilters.test.js`
Expected: PASS

- [ ] **Step 2: Build app**

Run: `wails build`
Expected: PASS

- [ ] **Step 3: Manual behavior check**

Verify that:
- search narrows novels by title
- rule dropdown narrows novels by website rule
- combining both filters works
- current selected novel is not forcibly changed just because filters hide it
