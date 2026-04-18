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
