export function filterNovels(novels, searchText, ruleId) {
  const normalizedSearch = String(searchText || '').trim().toLowerCase()
  const normalizedRuleId = String(ruleId || '').trim()

  return novels.filter((novel) => {
    const matchesSearch = normalizedSearch === '' ||
      String(novel?.title || '').toLowerCase().includes(normalizedSearch)
    const matchesRule = normalizedRuleId === '' || String(novel?.ruleId || '') === normalizedRuleId
    return matchesSearch && matchesRule
  })
}
