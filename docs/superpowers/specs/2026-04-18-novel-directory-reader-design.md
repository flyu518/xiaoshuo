# Novel Directory Reader Design

**Date:** 2026-04-18

## Goal

Add a simple reading flow that lets a user open a novel, inspect its chapter directory, and click any chapter to read the chapter text without re-entering the export flow.

## Current Problems

- The app already has chapter analysis and export, but it still treats the directory as an export-only preview.
- Users who just want to browse a novel have no dedicated reading view.
- Chapter data is effectively shared, but the UI does not expose a clean path from novel list to directory to chapter reading.
- The current export page is already crowded, so adding reading controls there would make it worse.

## Recommended Structure

Use three related views that share the same novel and chapter data:

1. **Novel**
- Keep the novel list, novel editor, and site binding controls here.
- Add a clear entry point such as "Open Directory".
- This is the place where a user picks the book they want to work with.

2. **Directory**
- Show the chapter directory for the currently selected novel.
- Allow searching, paging, and cached chapter markers.
- Clicking a chapter should open that chapter in the reading view.

3. **Reading**
- Show the text of the currently selected chapter.
- Provide `Previous Chapter`, `Next Chapter`, and `Back to Directory`.
- Reuse the same chapter list and chapter URLs already collected from directory analysis.

## Interaction Rules

- The novel selected in the Novel view becomes the current novel for directory and reading.
- Opening a directory should not discard export selections or analysis state.
- Clicking a chapter in the directory should load that chapter into the reading view.
- The reading view should keep the current novel context so the user can move forward/backward without reopening the directory.
- Directory analysis results, cached chapter markers, and chapter URLs should be reused rather than fetched separately for reading.

## Data Boundaries

- A novel remains the top-level owned object: title, catalog URL, rule binding, and saved preferences.
- A directory is a view of the novel's chapter list, not a separate data model.
- A chapter reading session is transient UI state: current chapter URL, current chapter index, loaded chapter content.
- Export should continue to use the same chapter list and selected chapter URLs that directory analysis already produced.

## Layout Guidance

- Keep the existing tabbed workspace.
- Add the new reading flow within the Novel area first, rather than creating a brand-new top-level tab immediately.
- Use a directory list panel and a reading panel side by side on desktop if space allows.
- On smaller screens, the directory and reading panels should stack vertically.
- Keep the reading view lightweight so it does not compete with the export workspace.

## Error Handling And UX

- If a novel has no directory loaded yet, show a clear prompt to open or analyze the directory.
- If a chapter fails to load, show the error in the reading area without breaking the rest of the novel context.
- If the user navigates to the next or previous chapter at the edge of the list, disable the unavailable action.
- If cached content exists for a chapter, prefer the cached content before refetching.

## Testing Focus

- Opening a novel shows its directory.
- Clicking a chapter opens the chapter reading view.
- Previous/next chapter navigation stays within the same novel.
- Returning from Reading to Directory preserves the current novel and chapter position.
- Export behavior remains unchanged and still uses the shared chapter data.

## Recommendation

Start with the Novel -> Directory -> Reading flow inside the novel workspace before trying to redesign the entire app around reading. That keeps the change focused, reuses existing chapter data, and avoids making the export workspace even more crowded.
