# Tabbed Workspace Design

**Date:** 2026-04-17

## Goal

Reduce the current single-page crowding by splitting the app into a small set of top-level tabs that match the real workflow: manage novels, export content, manage site rules, and clear caches.

## Current Problems

- The page mixes high-frequency tasks and low-frequency maintenance tools in one scroll.
- Export-specific controls compete for space with rule editing and cache management.
- The user has to visually scan unrelated sections even when only doing one job.
- As features have grown, the current layout is functionally complete but cognitively heavy.

## Recommended Structure

Use four top-level tabs:

1. **导出**
- Default tab on launch
- Contains the current export task controls, analysis, chapter selection, keyword highlight, progress, and export action
- This is the main day-to-day workspace

2. **小说**
- Contains the novel list, search, site filter, and novel editor
- Used when adding or adjusting saved books

3. **规则**
- Contains site rule management and rule editing
- Keeps advanced scraping controls away from the main export screen

4. **缓存**
- Contains cache list and batch cache clearing
- Maintenance only; should not compete with exporting for space

## Interaction Rules

- Current novel selection remains global across tabs
- Current rule selection remains global where relevant
- Switching tabs does not clear analysis results, chapter selection, or form edits
- Export tab should still allow selecting the active novel from a compact control at the top
- Tabs should not force a wizard flow; users can jump freely between them

## Layout Guidance

- Replace the always-visible left sidebar with a slimmer navigation rail or top tab row, depending on what fits best in the current design language
- Keep the visual style consistent with the existing warm card-based UI
- On desktop, tabs should maximize space for the active panel
- On smaller screens, tabs should stack cleanly and avoid deep nested layouts

## State Boundaries

- `activeTab` becomes a frontend-only UI state
- Novel list search/filter stays inside the 小说 tab
- Analysis result and chapter selection stay in export state, not tab state
- Cache selection state stays in the 缓存 tab but should persist while the app is open

## Error Handling And UX

- If no novel is selected, 导出 tab should show a clear empty-state prompt
- If tabs hide the currently selected item, no implicit re-selection should occur
- Success/error/progress messages stay globally visible near the top of the active workspace

## Testing Focus

- Switching tabs preserves current state
- Export flow still works end-to-end from the 导出 tab
- Novel editing still works from the 小说 tab
- Rule editing still works from the 规则 tab
- Cache clearing still works from the 缓存 tab

## Recommendation

Use tabs now rather than more accordions or collapsible sections. The tool has crossed the threshold where information architecture matters more than squeezing everything into one overview page.
