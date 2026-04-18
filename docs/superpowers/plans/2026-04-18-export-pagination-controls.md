# Export Pagination Controls Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move the export tab pagination controls next to the page indicator so chapter navigation is visible without extra scrolling.

**Architecture:** Keep the export tab layout as a two-column workspace. Update the chapter preview header so the page number and previous/next buttons live on the same row, and remove the duplicate bottom pagination bar. Keep the responsive collapse behavior unchanged so small screens still stack vertically.

**Tech Stack:** Vue 3, Wails, CSS Grid/Flexbox

---

### Task 1: Move pagination into the preview header

**Files:**
- Modify: `frontend/src/AppMain.vue`
- Modify: `frontend/src/style.css`

- [ ] **Step 1: Update the preview header markup**

Move the `上一页` and `下一页` buttons from the bottom of the chapter list into the chapter preview header, placing them after the page number text.

- [ ] **Step 2: Remove the duplicate bottom pagination bar**

Delete the bottom `actions compact` block that currently renders the previous/next buttons under the chapter list.

- [ ] **Step 3: Adjust header styling**

Add a compact inline layout for the page indicator and pagination buttons so the row stays readable on wide screens and wraps cleanly on narrow screens.

- [ ] **Step 4: Verify the frontend build**

Run: `npm run build`
Expected: build succeeds with no Vue template or CSS errors.
