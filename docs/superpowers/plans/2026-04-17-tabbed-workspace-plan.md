# Tabbed Workspace Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reorganize the crowded single-page Wails UI into a tabbed workspace with dedicated screens for exporting, novels, rules, and cache management.

**Architecture:** Add a small frontend-only `activeTab` state and split the current monolithic page into four tab panels that reuse the existing business logic and form state. Keep current selections, analysis results, and export controls global so tab switching is visual reorganization rather than a workflow reset.

**Tech Stack:** Vue 3, Vite, Wails

---

### Task 1: Tab state and navigation shell

**Files:**
- Modify: `D:/go/wails/xiaoshuo/frontend/src/AppMain.vue`
- Modify: `D:/go/wails/xiaoshuo/frontend/src/style.css`

- [ ] **Step 1: Add frontend tab state**

Add:
```js
const uiState = reactive({
  activeTab: 'export',
})
```

Add localized tab labels for `export`, `novels`, `rules`, `cache`.

- [ ] **Step 2: Render a top tab bar**

Create a reusable tab row near the hero/header area and highlight the active tab.

- [ ] **Step 3: Run build to catch syntax errors early**

Run: `npm run build`
Expected: PASS

### Task 2: Split current content into tab panels

**Files:**
- Modify: `D:/go/wails/xiaoshuo/frontend/src/AppMain.vue`
- Modify: `D:/go/wails/xiaoshuo/frontend/src/style.css`

- [ ] **Step 1: Move export task and preview into 导出 tab**

Keep:
- active novel selector
- analyze/export controls
- progress and messages
- chapter selection and preview

- [ ] **Step 2: Move novel list and novel editor into 小说 tab**

Keep:
- novel search and rule filter
- novel list
- novel editor form

- [ ] **Step 3: Move rule editor into 规则 tab**

Keep current rule selection and rule editor fields only.

- [ ] **Step 4: Move cache panel into 缓存 tab**

Keep current cache list and batch clearing actions only.

- [ ] **Step 5: Build frontend again**

Run: `npm run build`
Expected: PASS

### Task 3: State preservation and polish

**Files:**
- Modify: `D:/go/wails/xiaoshuo/frontend/src/AppMain.vue`
- Modify: `D:/go/wails/xiaoshuo/frontend/src/style.css`

- [ ] **Step 1: Preserve state across tabs**

Verify that active novel, rule, analysis, selection, and filters do not reset on tab switch.

- [ ] **Step 2: Improve responsive layout for tabs**

Ensure tab row wraps cleanly on narrow widths and active panel spacing remains comfortable.

- [ ] **Step 3: Final verification**

Run: `npm run build`
Expected: PASS

Run: `wails build`
Expected: PASS and output binary at `build/bin/xiaoshuo.exe`
