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
