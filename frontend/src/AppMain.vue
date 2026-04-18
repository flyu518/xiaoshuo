<script setup>
import { computed, onMounted, reactive, watch } from 'vue'
import { EventsOn } from '../wailsjs/runtime/runtime'
import * as Backend from '../wailsjs/go/main/App.js'
import { matchChapterByKeywords, parseKeywordInput } from './chapterFilters.js'
import { filterNovels } from './novelListFilters.js'
import { normalizeWorkspaceTab, workspaceTabs } from './workspaceTabs.js'
import { getSavedWorkspaceTab, saveWorkspaceTab } from './uiPreferences.js'

function hasWailsRuntime() {
  return typeof window !== 'undefined' && !!window.runtime && !!window.go?.main?.App
}

const state = reactive({
  loading: true,
  savingRule: false,
  savingNovel: false,
  analyzing: false,
  exporting: false,
  clearingCache: false,
  rules: [],
  novels: [],
  caches: [],
  analysis: null,
  progress: null,
  error: '',
  success: '',
  wailsReady: false,
})

const exportForm = reactive({
  novelId: '',
  maxChapters: 0,
  retryCount: 2,
  skipOnFailure: true,
  skipFilteredTitle: true,
  selectedChapterUrls: [],
  rangeInput: '',
  keywordInput: '',
  appliedKeywordInput: '',
  showMatchedOnly: false,
  showSelectedOnly: false,
  currentPage: 1,
  pageSize: 100,
})

const novelForm = reactive(createEmptyNovel())
const ruleForm = reactive(createEmptyRule())
const cacheForm = reactive({
  selectedNovelIds: [],
})
const listFilters = reactive({
  search: '',
  ruleId: '',
})
const uiState = reactive({
  activeTab: getSavedWorkspaceTab(),
  helpOpen: {},
})

let messageTimer = null

const selectedNovel = computed(() => state.novels.find((novel) => novel.id === exportForm.novelId) ?? null)
const selectedRule = computed(() => {
  const ruleID = novelForm.ruleId || selectedNovel.value?.ruleId || ''
  return state.rules.find((rule) => rule.id === ruleID) ?? null
})
const analyzedChapters = computed(() => state.analysis?.chapters ?? [])
const selectedChapterSet = computed(() => new Set(exportForm.selectedChapterUrls))
const selectedChapterCount = computed(() => exportForm.selectedChapterUrls.length)
const parsedKeywords = computed(() => parseKeywordInput(exportForm.appliedKeywordInput))
const matchedChapterUrls = computed(() => {
  const matched = new Set()
  for (const chapter of analyzedChapters.value) {
    if (matchChapterByKeywords(chapter, parsedKeywords.value)) {
      matched.add(chapter.url)
    }
  }
  return matched
})
const visibleChapters = computed(() => {
  return analyzedChapters.value.filter((chapter) => {
    if (exportForm.showMatchedOnly && !matchedChapterUrls.value.has(chapter.url)) {
      return false
    }
    if (exportForm.showSelectedOnly && !selectedChapterSet.value.has(chapter.url)) {
      return false
    }
    return true
  })
})
const totalPages = computed(() => {
  if (visibleChapters.value.length === 0) {
    return 1
  }
  return Math.max(1, Math.ceil(visibleChapters.value.length / exportForm.pageSize))
})
const pagedChapters = computed(() => {
  const startIndex = (exportForm.currentPage - 1) * exportForm.pageSize
  return visibleChapters.value.slice(startIndex, startIndex + exportForm.pageSize)
})
const currentPageAllSelected = computed(() => (
  pagedChapters.value.length > 0 &&
  pagedChapters.value.every((chapter) => selectedChapterSet.value.has(chapter.url))
))
const filteredNovels = computed(() => filterNovels(state.novels, listFilters.search, listFilters.ruleId))

const zh = {
  appTitle: '\u5c0f\u8bf4\u63d0\u53d6\u5668',
  appSubtitle: '\u628a\u7f51\u7ad9\u89c4\u5219\u3001\u5c0f\u8bf4\u4e0e\u5bfc\u51fa\u4efb\u52a1\u5206\u5f00\u7ba1\u7406\uff0c\u6293\u53d6\u66f4\u6e05\u6670\u3002',
  novels: '\u5c0f\u8bf4',
  rules: '\u7f51\u7ad9\u89c4\u5219',
  cache: '\u7f13\u5b58\u7ba1\u7406',
  novelSearch: '\u641c\u7d22\u5c0f\u8bf4',
  novelSearchPlaceholder: '\u6309\u4e66\u540d\u641c\u7d22',
  filterRule: '\u6309\u7ad9\u70b9\u7b5b\u9009',
  allRules: '\u5168\u90e8\u7ad9\u70b9',
  allChapters: '\u5168\u90e8',
  newNovel: '\u65b0\u5efa\u5c0f\u8bf4',
  newRule: '\u65b0\u5efa\u89c4\u5219',
  tabExport: '\u5bfc\u51fa',
  tabNovels: '\u5c0f\u8bf4',
  tabRules: '\u89c4\u5219',
  tabCache: '\u7f13\u5b58',
  task: '\u5bfc\u51fa\u4efb\u52a1',
  taskTitle: '\u4ece\u5f53\u524d\u5c0f\u8bf4\u51fa\u53d1\u5206\u6790\u76ee\u5f55\u5e76\u5bfc\u51fa',
  novel: '\u5f53\u524d\u5c0f\u8bf4',
  noNovelSelected: '\u672a\u9009\u62e9\u5c0f\u8bf4',
  novelTitle: '\u5c0f\u8bf4\u540d\u79f0',
  novelTitlePlaceholder: '\u4f8b\u5982\uff1a\u9752\u5c71',
  catalogUrl: '\u76ee\u5f55\u5730\u5740',
  catalogPlaceholder: '\u8f93\u5165\u8fd9\u672c\u5c0f\u8bf4\u7684\u76ee\u5f55 URL',
  boundRule: '\u7ed1\u5b9a\u89c4\u5219',
  chapterLimit: '\u5bfc\u51fa\u7ae0\u8282\u6570',
  chapterLimitPlaceholder: '0 \u8868\u793a\u5168\u90e8',
  retryCount: '\u5931\u8d25\u91cd\u8bd5\u6b21\u6570',
  retryPlaceholder: '\u5efa\u8bae 1-3 \u6b21',
  skipOnFailure: '\u5931\u8d25\u65f6\u8df3\u8fc7\u5e76\u8bb0\u5f55\u5230\u5c3e\u90e8',
  skipFiltered: '\u8df3\u8fc7\u201c\u8bf7\u5047 / \u603b\u7ed3 / \u611f\u8a00\u201d\u7b49\u6807\u9898\u5339\u914d\u7684\u7ae0\u8282',
  parsing: '\u5206\u6790\u4e2d...',
  parseCatalog: '\u5206\u6790\u76ee\u5f55',
  exporting: '\u5bfc\u51fa\u4e2d...',
  exportTxt: '\u5bfc\u51fa TXT',
  novelEditor: '\u5c0f\u8bf4\u7f16\u8f91',
  ruleEditor: '\u89c4\u5219\u7f16\u8f91',
  current: '\u5f53\u524d',
  novelId: '\u5c0f\u8bf4 ID',
  novelIdPlaceholder: '\u7559\u7a7a\u81ea\u52a8\u751f\u6210',
  saveNovel: '\u4fdd\u5b58\u5c0f\u8bf4',
  savingNovel: '\u4fdd\u5b58\u4e2d...',
  deleteNovel: '\u5220\u9664\u5c0f\u8bf4',
  ruleId: '\u89c4\u5219 ID',
  ruleIdPlaceholder: '\u7559\u7a7a\u81ea\u52a8\u751f\u6210',
  ruleName: '\u89c4\u5219\u540d\u79f0',
  ruleNamePlaceholder: '\u4f8b\u5982\uff1a\u5b99\u65af\u5c0f\u8bf4\u7f51',
  matchDomains: '\u5339\u914d\u57df\u540d',
  matchDomainsPlaceholder: '\u6bcf\u884c\u4e00\u4e2a\uff0c\u5982 zhswx.com',
  matchDomainsHelp: '\u7528\u6765\u5224\u65ad\u5f53\u524d\u5c0f\u8bf4\u76ee\u5f55\u5730\u5740\u5c5e\u4e8e\u54ea\u4e2a\u7ad9\u70b9\u89c4\u5219\u3002',
  catalogSelector: '\u76ee\u5f55\u7ae0\u8282\u9009\u62e9\u5668',
  catalogSelectorPlaceholder: '\u4f8b\u5982\uff1atd.chapterlist a[href^=\"/read/\"]',
  catalogSelectorHelp: '\u5728\u76ee\u5f55\u9875\u91cc\u7528\u6765\u627e\u6bcf\u4e00\u7ae0\u94fe\u63a5\u7684 CSS \u9009\u62e9\u5668\u3002',
  contentSelector: '\u6b63\u6587\u9009\u62e9\u5668',
  contentSelectorPlaceholder: '\u4f8b\u5982\uff1a#content \u6216 div.content',
  contentSelectorHelp: '\u5728\u7ae0\u8282\u9875\u91cc\u7528\u6765\u627e\u6b63\u6587\u5185\u5bb9\u533a\u57df\u7684 CSS \u9009\u62e9\u5668\u3002',
  headingText: '\u76ee\u5f55\u6807\u9898\u6587\u672c',
  headingPlaceholder: '\u53ef\u9009\uff0c\u5982\uff1a\u300a\u9752\u5c71\u300b\u6b63\u6587',
  headingHelp: '\u53ef\u9009\uff0c\u7528\u6765\u8f85\u52a9\u5b9a\u4f4d\u76ee\u5f55\u533a\u5757\uff0c\u4e0d\u586b\u4e5f\u53ef\u4ee5\u3002',
  sectionSelector: '\u76ee\u5f55\u533a\u5757\u9009\u62e9\u5668',
  sectionPlaceholder: '\u53ef\u9009\uff0c\u5982 div.section-box',
  sectionHelp: '\u53ef\u9009\uff0c\u5148\u9650\u5b9a\u76ee\u5f55\u6240\u5728\u533a\u57df\uff0c\u518d\u5728\u533a\u57df\u5185\u627e\u7ae0\u8282\u94fe\u63a5\u3002',
  nextPageSelector: '\u4e0b\u4e00\u9875\u9009\u62e9\u5668',
  nextPagePlaceholder: '\u4f8b\u5982\uff1aa:contains(\"\u4e0b\u4e00\u9875\")',
  nextPageHelp: '\u5982\u679c\u4e00\u7ae0\u5206\u6210\u591a\u9875\uff0c\u7528\u5b83\u627e\u5230\u201c\u4e0b\u4e00\u9875\u201d\u6309\u94ae\u3002',
  nextChapterSelector: '\u4e0b\u4e00\u7ae0\u9009\u62e9\u5668',
  nextChapterPlaceholder: '\u4f8b\u5982\uff1aa:contains(\"\u4e0b\u4e00\u7ae0\")',
  nextChapterHelp: '\u53ef\u9009\uff0c\u7528\u6765\u627e\u5230\u201c\u4e0b\u4e00\u7ae0\u201d\u6309\u94ae\uff0c\u9002\u5408\u4ece\u7ae0\u8282\u9875\u987a\u7740\u6293\u53d6\u3002',
  cleanupSelectors: '\u6e05\u7406\u9009\u62e9\u5668',
  cleanupSelectorsPlaceholder: '\u6bcf\u884c\u4e00\u4e2a\uff0c\u4f8b\u5982\uff1a.ad\\nscript',
  cleanupSelectorsHelp: '\u6293\u6b63\u6587\u65f6\u5148\u5220\u6389\u8fd9\u4e9b HTML \u5143\u7d20\uff0c\u5e38\u7528\u6765\u53bb\u5e7f\u544a\u3001\u63d0\u793a\u6761\u3001\u6309\u94ae\u533a\u57df\u3002',
  stopTexts: '\u6b63\u6587\u505c\u6b62\u6587\u672c',
  stopTextsPlaceholder: '\u6bcf\u884c\u4e00\u4e2a\uff0c\u4f8b\u5982\uff1a\u624b\u673a\u9605\u8bfb\u8bf7\u8bbf\u95ee',
  stopTextsHelp: '\u5982\u679c\u6b63\u6587\u91cc\u6df7\u5165\u4e86\u56fa\u5b9a\u63d0\u793a\u6587\u5b57\uff0c\u53ef\u4ee5\u7528\u8fd9\u4e9b\u5185\u5bb9\u505c\u6b62\u6216\u622a\u65ad\u3002',
  removeMatchingLines: '\u5220\u9664\u5339\u914d\u884c',
  removeMatchingLinesPlaceholder: '\u6bcf\u884c\u4e00\u4e2a\uff0c\u53ea\u8981\u8fd9\u4e00\u884c\u5305\u542b\u5b83\u5c31\u6574\u884c\u5220\u6389',
  removeMatchingLinesHelp: '\u9002\u5408\u5904\u7406\u6ca1\u6709\u7279\u6b8a HTML \u6807\u7b7e\u7684\u56fa\u5b9a\u63d0\u793a\u8bed\uff0c\u53ea\u8981\u5f53\u524d\u884c\u5305\u542b\u5339\u914d\u6587\u5b57\u5c31\u6574\u884c\u5220\u9664\u3002',
  textReplaceRules: '\u6587\u672c\u66ff\u6362',
  textReplaceRulesHelp: '\u53ef\u4ee5\u628a\u56fa\u5b9a\u6587\u5b57\u66ff\u6362\u6210\u65b0\u6587\u5b57\uff0c\u5982\u679c\u201c\u66ff\u6362\u4e3a\u201d\u7559\u7a7a\uff0c\u5c31\u76f8\u5f53\u4e8e\u5220\u9664\u8fd9\u6bb5\u56fa\u5b9a\u6587\u5b57\u3002',
  regexRules: '\u9ad8\u7ea7\u6b63\u5219\u89c4\u5219',
  regexRulesHelp: '\u9002\u5408\u5904\u7406\u66f4\u590d\u6742\u7684\u6587\u5b57\u89c4\u5219\uff0c\u6bd4\u5982\u6b63\u5219\u5220\u884c\u3001\u6b63\u5219\u66ff\u6362\u3002\u4e0d\u9700\u8981\u65f6\u53ef\u4ee5\u4e0d\u586b\u3002',
  addRule: '\u65b0\u589e\u89c4\u5219',
  matchText: '\u5339\u914d\u6587\u672c',
  replaceText: '\u66ff\u6362\u4e3a',
  regexPattern: '\u6b63\u5219',
  ruleEnabled: '\u542f\u7528',
  caseSensitive: '\u533a\u5206\u5927\u5c0f\u5199',
  replaceFirst: '\u4ec5\u5904\u7406\u9996\u4e2a\u5339\u914d',
  removeLine: '\u5220\u9664\u5339\u914d\u884c',
  replacePlaceholder: '\u7559\u7a7a\u5219\u5220\u9664\u5339\u914d\u6587\u5b57',
  regexPlaceholder: '\u4f8b\u5982\uff1a\u4f5c\u8005\u63d0\u793a\uff1a.*',
  skipRegex: '\u8df3\u8fc7\u6807\u9898\u6b63\u5219',
  skipRegexPlaceholder: '\u6bcf\u884c\u4e00\u4e2a\uff0c\u4f8b\u5982\uff1a\u8bf7\u5047\\n\u611f\u8a00',
  skipRegexHelp: '\u7ae0\u8282\u6807\u9898\u5982\u679c\u5339\u914d\u8fd9\u4e9b\u6b63\u5219\uff0c\u5bfc\u51fa\u65f6\u53ef\u4ee5\u8df3\u8fc7\u3002',
  notes: '\u5907\u6ce8',
  notesPlaceholder: '\u8bb0\u5f55\u7ad9\u70b9\u7279\u6b8a\u89c4\u5219',
  saveRule: '\u4fdd\u5b58\u89c4\u5219',
  savingRule: '\u4fdd\u5b58\u4e2d...',
  deleteRule: '\u5220\u9664\u89c4\u5219',
  preview: '\u76ee\u5f55\u9884\u89c8',
  chapters: '\u7ae0',
  selectedChapters: '\u5df2\u9009\u7ae0\u8282',
  rangeInput: '\u7ae0\u8282\u8303\u56f4',
  rangePlaceholder: '\u4f8b\u5982\uff1a1-200,520-560',
  applyRange: '\u6309\u8303\u56f4\u9009\u4e2d',
  rangeHint: '\u8f93\u5165\u8303\u56f4\u540e\u56de\u8f66\u6216\u5931\u53bb\u7126\u70b9\u4f1a\u81ea\u52a8\u5e94\u7528\uff0c\u4e5f\u53ef\u70b9\u4e0b\u9762\u6309\u94ae\u3002',
  conditionSelection: '\u6761\u4ef6\u9009\u62e9',
  actionSelection: '\u901a\u7528\u64cd\u4f5c',
  pageActions: '\u672c\u9875\u64cd\u4f5c',
  viewFilter: '\u89c6\u56fe\u7b5b\u9009',
  rangeInvalid: '\u8bf7\u8f93\u5165\u6709\u6548\u7684\u7ae0\u8282\u8303\u56f4\uff0c\u4f8b\u5982 1-200\u3002',
  rangeApplied: '\u5df2\u6309\u7ae0\u8282\u8303\u56f4\u9009\u4e2d {count} \u7ae0\u3002',
  keywordInput: '\u5173\u952e\u8bcd\u9ad8\u4eae',
  keywordPlaceholder: '\u4f8b\u5982\uff1a\u8bf7\u5047,\u611f\u8a00,\u756a\u5916',
  keywordApplied: '\u5df2\u5e94\u7528\u5173\u952e\u8bcd\u6761\u4ef6\u3002',
  noMatchedOnPage: '\u5f53\u524d\u9875\u6ca1\u6709\u547d\u4e2d\u5173\u952e\u8bcd\u7684\u7ae0\u8282\u3002',
  matchedSelectedOnPage: '\u5df2\u5728\u5f53\u524d\u9875\u52fe\u9009 {count} \u4e2a\u547d\u4e2d\u7ae0\u8282\u3002',
  matchedClearedOnPage: '\u5df2\u5728\u5f53\u524d\u9875\u53d6\u6d88 {count} \u4e2a\u547d\u4e2d\u7ae0\u8282\u3002',
  showMatchedOnly: '\u4ec5\u770b\u5173\u952e\u5b57\u547d\u4e2d\u7ae0\u8282',
  showSelectedOnly: '\u4ec5\u770b\u5df2\u9009\u7ae0\u8282',
  selectMatchedOnPage: '\u672c\u9875\u52fe\u9009\u5339\u914d',
  clearMatchedOnPage: '\u672c\u9875\u53d6\u6d88\u5339\u914d',
  matchedBadge: '\u5339\u914d',
  selectAll: '\u5168\u9009',
  clearSelection: '\u6e05\u7a7a',
  invertSelection: '\u53cd\u9009',
  currentPageToggle: '\u672c\u9875\u5168\u9009/\u53d6\u6d88',
  previousPage: '\u4e0a\u4e00\u9875',
  nextPage: '\u4e0b\u4e00\u9875',
  pageStatus: '\u9875\u7801',
  noSelection: '\u8bf7\u81f3\u5c11\u9009\u62e9\u4e00\u7ae0\u518d\u5bfc\u51fa\u3002',
  previewEmpty: '\u70b9\u4e00\u6b21\u201c\u5206\u6790\u76ee\u5f55\u201d\uff0c\u8fd9\u91cc\u4f1a\u663e\u793a\u8bc6\u522b\u5230\u7684\u7ae0\u8282\u5217\u8868\u3002',
  exportSummary: '\u5bfc\u51fa\u6982\u89c8',
  novelsEmpty: '\u8fd8\u6ca1\u6709\u4fdd\u5b58\u7684\u5c0f\u8bf4\uff0c\u53ef\u4ee5\u5148\u65b0\u5efa\u4e00\u672c\u3002',
  novelsFilteredEmpty: '\u5f53\u524d\u641c\u7d22\u6216\u7b5b\u9009\u6761\u4ef6\u4e0b\u6ca1\u6709\u5c0f\u8bf4\u3002',
  rulesEmpty: '\u8fd8\u6ca1\u6709\u89c4\u5219\uff0c\u53ef\u4ee5\u5148\u65b0\u5efa\u4e00\u6761\u3002',
  cacheEmpty: '\u5f53\u524d\u6ca1\u6709\u53ef\u7528\u7f13\u5b58',
  clearCache: '\u6e05\u7406\u9009\u4e2d\u7f13\u5b58',
  clearingCache: '\u6e05\u7406\u4e2d...',
  cacheFiles: '\u7ae0\u8282\u7f13\u5b58',
  cacheSize: '\u5360\u7528\u7a7a\u95f4',
  cachedBadge: '\u5df2\u7f13\u5b58',
  browserMode: '\u5f53\u524d\u662f\u666e\u901a\u6d4f\u89c8\u5668\u9884\u89c8\uff0c\u8bf7\u5728\u9879\u76ee\u6839\u76ee\u5f55\u8fd0\u884c wails dev \u6253\u5f00\u684c\u9762\u7248\u3002',
}

const exportSummary = computed(() => [
  { label: zh.selectedChapters, value: `${selectedChapterCount.value} / ${analyzedChapters.value.length || 0}` },
  { label: zh.chapterLimit, value: Number(exportForm.maxChapters) > 0 ? String(exportForm.maxChapters) : zh.allChapters },
])

onMounted(async () => {
  const ready = await waitForWailsRuntime()
  state.wailsReady = ready

  if (!ready) {
    state.loading = false
    state.error = zh.browserMode
    return
  }

  await refreshState()
  EventsOn('crawl-progress', (payload) => {
    state.progress = payload
  })
})

async function waitForWailsRuntime(maxWaitMs = 4000) {
  const startedAt = Date.now()
  while (Date.now() - startedAt < maxWaitMs) {
    if (hasWailsRuntime()) {
      return true
    }
    await new Promise((resolve) => setTimeout(resolve, 100))
  }
  return hasWailsRuntime()
}

watch(() => exportForm.showMatchedOnly, ensurePageInRange)
watch(() => exportForm.showSelectedOnly, ensurePageInRange)
watch(
  () => [state.error, state.success],
  ([error, success]) => {
    if (messageTimer) {
      window.clearTimeout(messageTimer)
      messageTimer = null
    }
    const message = error || success
    if (!message) {
      return
    }
    messageTimer = window.setTimeout(() => {
      state.error = ''
      state.success = ''
      messageTimer = null
    }, 2800)
  }
)
watch(() => exportForm.appliedKeywordInput, ensurePageInRange)
watch(() => exportForm.selectedChapterUrls, ensurePageInRange)
watch(() => uiState.activeTab, (tab) => saveWorkspaceTab(tab))

function createEmptyNovel() {
  return {
    id: '',
    title: '',
    catalogUrl: '',
    ruleId: '',
  }
}

function createEmptyRule() {
  return {
    id: '',
    name: '',
    matchDomains: 'example.com',
    catalogSectionHeadingText: '',
    catalogSectionContainer: '',
    catalogChapterLinkSelector: '',
    chapterTitleSelector: '',
    chapterContentSelector: '',
    nextPageSelector: '',
    nextChapterSelector: '',
    contentCleanupSelectors: '',
    contentStopTexts: '',
    removeMatchingLines: '',
    textReplacementRules: [createTextReplacementRule()],
    regexReplacementRules: [createRegexReplacementRule()],
    skipChapterTitlePatterns: '',
    notes: '',
  }
}

function createTextReplacementRule() {
  return {
    match: '',
    replace: '',
    caseSensitive: false,
    replaceFirst: false,
    enabled: true,
  }
}

function createRegexReplacementRule() {
  return {
    pattern: '',
    replace: '',
    removeLine: false,
    replaceFirst: false,
    enabled: true,
  }
}

function splitLines(text) {
  return text.split('\n').map((item) => item.trim()).filter(Boolean)
}

function buildNovelPayload() {
  return {
    id: novelForm.id.trim(),
    title: novelForm.title.trim(),
    catalogUrl: novelForm.catalogUrl.trim(),
    ruleId: novelForm.ruleId,
  }
}

function buildRulePayload() {
  return {
    id: ruleForm.id.trim(),
    name: ruleForm.name.trim(),
    matchDomains: splitLines(ruleForm.matchDomains),
    catalogSectionHeadingText: ruleForm.catalogSectionHeadingText.trim(),
    catalogSectionContainer: ruleForm.catalogSectionContainer.trim(),
    catalogChapterLinkSelector: ruleForm.catalogChapterLinkSelector.trim(),
    chapterTitleSelector: ruleForm.chapterTitleSelector.trim(),
    chapterContentSelector: ruleForm.chapterContentSelector.trim(),
    nextPageSelector: ruleForm.nextPageSelector.trim(),
    nextChapterSelector: ruleForm.nextChapterSelector.trim(),
    contentCleanupSelectors: splitLines(ruleForm.contentCleanupSelectors),
    contentStopTexts: splitLines(ruleForm.contentStopTexts),
    removeMatchingLines: splitLines(ruleForm.removeMatchingLines),
    textReplacementRules: ruleForm.textReplacementRules
      .map((rule) => ({
        match: rule.match.trim(),
        replace: rule.replace.trim(),
        caseSensitive: !!rule.caseSensitive,
        replaceFirst: !!rule.replaceFirst,
        enabled: !!rule.enabled,
      }))
      .filter((rule) => rule.match),
    regexReplacementRules: ruleForm.regexReplacementRules
      .map((rule) => ({
        pattern: rule.pattern.trim(),
        replace: rule.replace.trim(),
        removeLine: !!rule.removeLine,
        replaceFirst: !!rule.replaceFirst,
        enabled: !!rule.enabled,
      }))
      .filter((rule) => rule.pattern),
    skipChapterTitlePatterns: splitLines(ruleForm.skipChapterTitlePatterns),
    notes: ruleForm.notes.trim(),
  }
}

function applyNovelToForm(novel) {
  Object.assign(novelForm, {
    id: novel.id ?? '',
    title: novel.title ?? '',
    catalogUrl: novel.catalogUrl ?? '',
    ruleId: novel.ruleId ?? '',
  })
}

function applyRuleToForm(rule) {
  Object.assign(ruleForm, {
    id: rule.id ?? '',
    name: rule.name ?? '',
    matchDomains: (rule.matchDomains ?? []).join('\n'),
    catalogSectionHeadingText: rule.catalogSectionHeadingText ?? '',
    catalogSectionContainer: rule.catalogSectionContainer ?? '',
    catalogChapterLinkSelector: rule.catalogChapterLinkSelector ?? '',
    chapterTitleSelector: rule.chapterTitleSelector ?? '',
    chapterContentSelector: rule.chapterContentSelector ?? '',
    nextPageSelector: rule.nextPageSelector ?? '',
    nextChapterSelector: rule.nextChapterSelector ?? '',
    contentCleanupSelectors: (rule.contentCleanupSelectors ?? []).join('\n'),
    contentStopTexts: (rule.contentStopTexts ?? []).join('\n'),
    removeMatchingLines: (rule.removeMatchingLines ?? []).join('\n'),
    textReplacementRules: (rule.textReplacementRules?.length ? rule.textReplacementRules : [createTextReplacementRule()]).map((item) => ({
      match: item.match ?? '',
      replace: item.replace ?? '',
      caseSensitive: !!item.caseSensitive,
      replaceFirst: !!item.replaceFirst,
      enabled: item.enabled !== false,
    })),
    regexReplacementRules: (rule.regexReplacementRules?.length ? rule.regexReplacementRules : [createRegexReplacementRule()]).map((item) => ({
      pattern: item.pattern ?? '',
      replace: item.replace ?? '',
      removeLine: !!item.removeLine,
      replaceFirst: !!item.replaceFirst,
      enabled: item.enabled !== false,
    })),
    skipChapterTitlePatterns: (rule.skipChapterTitlePatterns ?? []).join('\n'),
    notes: rule.notes ?? '',
  })
}

function resetNovelForm() {
  Object.assign(novelForm, createEmptyNovel())
}

function resetRuleForm() {
  Object.assign(ruleForm, createEmptyRule())
}

function addTextReplacementRule() {
  ruleForm.textReplacementRules.push(createTextReplacementRule())
}

function removeTextReplacementRule(index) {
  ruleForm.textReplacementRules.splice(index, 1)
  if (ruleForm.textReplacementRules.length === 0) {
    addTextReplacementRule()
  }
}

function addRegexReplacementRule() {
  ruleForm.regexReplacementRules.push(createRegexReplacementRule())
}

function removeRegexReplacementRule(index) {
  ruleForm.regexReplacementRules.splice(index, 1)
  if (ruleForm.regexReplacementRules.length === 0) {
    addRegexReplacementRule()
  }
}

function normalizeSelection(urls) {
  return [...new Set(urls.map((url) => String(url || '').trim()).filter(Boolean))]
}

function resetChapterSelection(chapters = analyzedChapters.value) {
  exportForm.selectedChapterUrls = normalizeSelection(chapters.map((chapter) => chapter.url))
  exportForm.rangeInput = ''
  exportForm.keywordInput = ''
  exportForm.appliedKeywordInput = ''
  exportForm.showMatchedOnly = false
  exportForm.showSelectedOnly = false
  exportForm.currentPage = 1
}

function selectAllChapters() {
  resetChapterSelection()
}

function clearChapterSelection() {
  exportForm.selectedChapterUrls = []
}

function invertChapterSelection() {
  const selected = selectedChapterSet.value
  exportForm.selectedChapterUrls = analyzedChapters.value
    .map((chapter) => chapter.url)
    .filter((url) => !selected.has(url))
}

function toggleCurrentPageSelection() {
  const next = new Set(exportForm.selectedChapterUrls)
  if (currentPageAllSelected.value) {
    for (const chapter of pagedChapters.value) {
      next.delete(chapter.url)
    }
  } else {
    for (const chapter of pagedChapters.value) {
      next.add(chapter.url)
    }
  }
  exportForm.selectedChapterUrls = [...next]
}

function toggleChapterSelection(chapterURL, checked) {
  const next = new Set(exportForm.selectedChapterUrls)
  if (checked) {
    next.add(chapterURL)
  } else {
    next.delete(chapterURL)
  }
  exportForm.selectedChapterUrls = [...next]
}

function parseRangeInput(input, maxCount) {
  const selectedIndexes = new Set()
  for (const segment of String(input || '').split(',')) {
    const value = segment.trim()
    if (!value) {
      continue
    }

    if (/^\d+$/.test(value)) {
      const index = Number(value)
      if (index >= 1 && index <= maxCount) {
        selectedIndexes.add(index - 1)
      }
      continue
    }

    const match = value.match(/^(\d+)\s*-\s*(\d+)$/)
    if (!match) {
      continue
    }

    let start = Number(match[1])
    let end = Number(match[2])
    if (start > end) {
      ;[start, end] = [end, start]
    }
    start = Math.max(1, start)
    end = Math.min(maxCount, end)
    for (let index = start; index <= end; index += 1) {
      selectedIndexes.add(index - 1)
    }
  }
  return [...selectedIndexes].sort((a, b) => a - b)
}

function applyRangeSelection(options = {}) {
  const { silentEmpty = false } = options
  const rawRange = exportForm.rangeInput.trim()
  if (!rawRange) {
    if (!silentEmpty) {
      state.error = zh.rangeInvalid
      state.success = ''
    } else if (state.error === zh.rangeInvalid) {
      state.error = ''
    }
    return
  }

  const indexes = parseRangeInput(rawRange, analyzedChapters.value.length)
  if (indexes.length === 0) {
    state.error = zh.rangeInvalid
    state.success = ''
    return
  }

  exportForm.selectedChapterUrls = indexes.map((index) => analyzedChapters.value[index]?.url).filter(Boolean)
  exportForm.currentPage = (exportForm.showMatchedOnly || exportForm.showSelectedOnly)
    ? 1
    : Math.floor(indexes[0] / exportForm.pageSize) + 1
  state.error = ''
  state.success = zh.rangeApplied.replace('{count}', String(indexes.length))
}

function commitRangeSelection() {
  applyRangeSelection({ silentEmpty: true })
}

function applyKeywordSelection() {
  exportForm.appliedKeywordInput = exportForm.keywordInput.trim()
  exportForm.currentPage = 1
}

function commitKeywordSelection() {
  applyKeywordSelection()
  state.error = ''
  if (exportForm.appliedKeywordInput) {
    state.success = zh.keywordApplied
  }
}

function selectMatchedOnCurrentPage() {
  if (exportForm.keywordInput.trim() !== exportForm.appliedKeywordInput.trim()) {
    applyKeywordSelection()
  }

  const matchedOnPage = pagedChapters.value.filter((chapter) => matchedChapterUrls.value.has(chapter.url))
  if (matchedOnPage.length === 0) {
    state.error = zh.noMatchedOnPage
    state.success = ''
    return
  }

  const next = new Set(exportForm.selectedChapterUrls)
  let changed = 0
  for (const chapter of matchedOnPage) {
    if (!next.has(chapter.url)) {
      next.add(chapter.url)
      changed += 1
    }
  }
  exportForm.selectedChapterUrls = [...next]
  state.error = ''
  state.success = zh.matchedSelectedOnPage.replace('{count}', String(changed))
}

function clearMatchedOnCurrentPage() {
  if (exportForm.keywordInput.trim() !== exportForm.appliedKeywordInput.trim()) {
    applyKeywordSelection()
  }

  const matchedOnPage = pagedChapters.value.filter((chapter) => matchedChapterUrls.value.has(chapter.url))
  if (matchedOnPage.length === 0) {
    state.error = zh.noMatchedOnPage
    state.success = ''
    return
  }

  const next = new Set(exportForm.selectedChapterUrls)
  let changed = 0
  for (const chapter of matchedOnPage) {
    if (next.has(chapter.url)) {
      next.delete(chapter.url)
      changed += 1
    }
  }
  exportForm.selectedChapterUrls = [...next]
  state.error = ''
  state.success = zh.matchedClearedOnPage.replace('{count}', String(changed))
}

function setPage(page) {
  exportForm.currentPage = Math.min(totalPages.value, Math.max(1, page))
}

function ensurePageInRange() {
  if (exportForm.currentPage > totalPages.value) {
    exportForm.currentPage = totalPages.value
  }
  if (exportForm.currentPage < 1) {
    exportForm.currentPage = 1
  }
}

async function refreshState() {
  if (!hasWailsRuntime()) {
    state.loading = false
    return
  }

  state.loading = true
  try {
    const result = await Backend.LoadState()
    state.rules = result.rules ?? []
    state.novels = result.novels ?? []
    state.caches = await Backend.ListNovelCaches()

    if (!exportForm.novelId && state.novels.length > 0) {
      pickNovel(state.novels[0])
    } else if (!selectedRule.value && state.rules.length > 0) {
      applyRuleToForm(state.rules[0])
    }
  } catch (error) {
    state.error = error?.message ?? String(error)
  } finally {
    state.loading = false
  }
}

function pickNovel(novel) {
  if (!novel) {
    return
  }
  syncSelectedNovel(novel, { resetAnalysis: true, clearFeedback: true })
}

function pickRule(rule) {
  if (!rule) {
    return
  }
  novelForm.ruleId = rule.id
  applyRuleToForm(rule)
  state.error = ''
  state.success = ''
}

function syncSelectedNovel(novel, options = {}) {
  if (!novel) {
    return
  }

  const {
    resetAnalysis = false,
    clearFeedback = false,
  } = options

  exportForm.novelId = novel.id
  applyNovelToForm(novel)

  const rule = state.rules.find((item) => item.id === novel.ruleId)
  if (rule) {
    applyRuleToForm(rule)
  }

  if (resetAnalysis) {
    state.analysis = null
    clearChapterSelection()
    exportForm.currentPage = 1
  }

  if (clearFeedback) {
    state.error = ''
    state.success = ''
  }
}

function handleNovelChange() {
  pickNovel(state.novels.find((novel) => novel.id === exportForm.novelId) ?? null)
}

function handleNovelRuleChange() {
  pickRule(state.rules.find((rule) => rule.id === novelForm.ruleId) ?? null)
}

async function saveCurrentForms() {
  const rulePayload = buildRulePayload()
  const ruleState = await Backend.SaveRule(rulePayload)
  state.rules = ruleState.rules ?? []
  state.novels = ruleState.novels ?? state.novels

  const savedRule = state.rules.find((rule) => rule.id === rulePayload.id) ?? state.rules.at(-1)
  if (savedRule) {
    applyRuleToForm(savedRule)
    novelForm.ruleId = savedRule.id
  }

  const novelPayload = buildNovelPayload()
  if (!novelPayload.ruleId && savedRule) {
    novelPayload.ruleId = savedRule.id
  }

  const novelState = await Backend.SaveNovel(novelPayload)
  state.rules = novelState.rules ?? state.rules
  state.novels = novelState.novels ?? []

  const savedNovel = state.novels.find((novel) => novel.id === novelPayload.id) ?? state.novels.at(-1)
  if (savedNovel) {
    syncSelectedNovel(savedNovel)
  }

  return { savedRule, savedNovel }
}

async function analyzeCatalog() {
  if (!hasWailsRuntime()) {
    state.error = zh.browserMode
    return
  }

  state.error = ''
  state.success = ''
  state.analysis = null
  state.analyzing = true
  try {
    const { savedNovel } = await saveCurrentForms()
    state.analysis = await Backend.AnalyzeCatalog({
      novelId: savedNovel?.id ?? exportForm.novelId,
    })
    resetChapterSelection(state.analysis.chapters ?? [])
    state.success = `\u76ee\u5f55\u5206\u6790\u5b8c\u6210\uff0c\u5171\u8bc6\u522b ${state.analysis.chapterCount} \u7ae0\u3002`
  } catch (error) {
    state.error = error?.message ?? String(error)
  } finally {
    state.analyzing = false
  }
}

async function saveNovel() {
  if (!hasWailsRuntime()) {
    state.error = zh.browserMode
    return
  }

  state.error = ''
  state.success = ''
  state.savingNovel = true
  try {
    const payload = buildNovelPayload()
    const result = await Backend.SaveNovel(payload)
    state.rules = result.rules ?? state.rules
    state.novels = result.novels ?? []
    const savedNovel = state.novels.find((novel) => novel.id === payload.id) ?? state.novels.at(-1)
    if (savedNovel) {
      syncSelectedNovel(savedNovel, { clearFeedback: true })
    }
    state.success = '\u5c0f\u8bf4\u5df2\u4fdd\u5b58\u3002'
  } catch (error) {
    state.error = error?.message ?? String(error)
  } finally {
    state.savingNovel = false
  }
}

async function removeNovel() {
  if (!hasWailsRuntime()) {
    state.error = zh.browserMode
    return
  }
  if (!novelForm.id) {
    return
  }

  state.error = ''
  state.success = ''
  try {
    const result = await Backend.DeleteNovel(novelForm.id)
    state.rules = result.rules ?? state.rules
    state.novels = result.novels ?? []
    if (exportForm.novelId === novelForm.id) {
      exportForm.novelId = state.novels[0]?.id ?? ''
    }
    if (state.novels.length > 0) {
      pickNovel(state.novels[0])
    } else {
      resetNovelForm()
      if (state.rules.length > 0) {
        applyRuleToForm(state.rules[0])
      } else {
        resetRuleForm()
      }
    }
    state.success = '\u5c0f\u8bf4\u5df2\u5220\u9664\u3002'
  } catch (error) {
    state.error = error?.message ?? String(error)
  }
}

async function saveRule() {
  if (!hasWailsRuntime()) {
    state.error = zh.browserMode
    return
  }

  state.error = ''
  state.success = ''
  state.savingRule = true
  try {
    const payload = buildRulePayload()
    const result = await Backend.SaveRule(payload)
    state.rules = result.rules ?? []
    state.novels = result.novels ?? state.novels
    const savedRule = state.rules.find((rule) => rule.id === payload.id) ?? state.rules.at(-1)
    if (savedRule) {
      pickRule(savedRule)
    }
    state.success = '\u89c4\u5219\u5df2\u4fdd\u5b58\u3002'
  } catch (error) {
    state.error = error?.message ?? String(error)
  } finally {
    state.savingRule = false
  }
}

async function removeRule() {
  if (!hasWailsRuntime()) {
    state.error = zh.browserMode
    return
  }
  if (!ruleForm.id) {
    return
  }

  state.error = ''
  state.success = ''
  try {
    const result = await Backend.DeleteRule(ruleForm.id)
    state.rules = result.rules ?? []
    state.novels = result.novels ?? state.novels
    state.caches = await Backend.ListNovelCaches()

    const nextRule = state.rules.find((rule) => rule.id === novelForm.ruleId) ?? state.rules[0]
    if (nextRule) {
      applyRuleToForm(nextRule)
      novelForm.ruleId = nextRule.id
    } else {
      resetRuleForm()
      novelForm.ruleId = ''
    }
    state.success = '\u89c4\u5219\u5df2\u5220\u9664\u3002'
  } catch (error) {
    state.error = error?.message ?? String(error)
  }
}

async function exportNovel() {
  if (!hasWailsRuntime()) {
    state.error = zh.browserMode
    return
  }

  state.error = ''
  state.success = ''
  state.exporting = true
  state.progress = null
  try {
    if (exportForm.selectedChapterUrls.length === 0) {
      throw new Error(zh.noSelection)
    }
    const { savedNovel } = await saveCurrentForms()
    const result = await Backend.ExportNovelAdvanced({
      novelId: savedNovel?.id ?? exportForm.novelId,
      maxChapters: Number(exportForm.maxChapters) || 0,
      retryCount: Number(exportForm.retryCount) || 0,
      skipOnFailure: exportForm.skipOnFailure,
      skipFilteredTitle: exportForm.skipFilteredTitle,
      selectedChapterUrls: [...exportForm.selectedChapterUrls],
    })
    state.caches = await Backend.ListNovelCaches()
    state.success = `\u5df2\u5bfc\u51fa ${result.exportedCount} \u7ae0\u5230 ${result.filePath}` +
      (result.failureCount ? `\uff0c\u53e6\u6709 ${result.failureCount} \u7ae0\u5df2\u8df3\u8fc7` : '')
  } catch (error) {
    state.error = error?.message ?? String(error)
  } finally {
    state.exporting = false
  }
}

async function clearSelectedCaches() {
  if (!hasWailsRuntime()) {
    state.error = zh.browserMode
    return
  }
  if (cacheForm.selectedNovelIds.length === 0) {
    return
  }

  state.error = ''
  state.success = ''
  state.clearingCache = true
  try {
    state.caches = await Backend.ClearNovelCaches(cacheForm.selectedNovelIds)
    cacheForm.selectedNovelIds = []
    state.success = '\u5df2\u6e05\u7406\u6240\u9009\u5c0f\u8bf4\u7684\u7f13\u5b58\u3002'
  } catch (error) {
    state.error = error?.message ?? String(error)
  } finally {
    state.clearingCache = false
  }
}

function formatBytes(value) {
  const size = Number(value) || 0
  if (size < 1024) {
    return `${size} B`
  }
  if (size < 1024 * 1024) {
    return `${(size / 1024).toFixed(1)} KB`
  }
  return `${(size / (1024 * 1024)).toFixed(1)} MB`
}

function tabLabel(tab) {
  const labels = {
    export: zh.tabExport,
    novels: zh.tabNovels,
    rules: zh.tabRules,
    cache: zh.tabCache,
  }
  return labels[tab] ?? zh.tabExport
}

function setActiveTab(tab) {
  uiState.activeTab = normalizeWorkspaceTab(tab)
}

function toggleHelp(key) {
  uiState.helpOpen[key] = !uiState.helpOpen[key]
}
</script>

<template>
  <div class="shell">
    <main class="content content-wide">
      <section v-if="!state.wailsReady" class="panel" style="margin-bottom: 20px;">
        <p class="message error" style="margin: 0;">{{ zh.browserMode }}</p>
      </section>

      <section class="panel tab-bar">
        <button
          v-for="tab in workspaceTabs"
          :key="tab"
          type="button"
          class="tab-pill"
          :class="{ active: uiState.activeTab === tab }"
          @click="setActiveTab(tab)"
        >
          {{ tabLabel(tab) }}
        </button>
      </section>

      <section v-if="state.progress" class="panel progress-strip">
        <strong>{{ state.progress.message }}</strong>
        <span v-if="state.progress.chapterTitle">{{ state.progress.chapterTitle }}</span>
        <span>{{ state.progress.current }}/{{ state.progress.total }}</span>
      </section>

      <section v-if="state.error || state.success" class="message-stack">
        <p v-if="state.error" class="message error">{{ state.error }}</p>
        <p v-if="state.success" class="message success">{{ state.success }}</p>
      </section>

      <section class="workspace-scroll">
      <template v-if="uiState.activeTab === 'export'">
        <section v-if="state.novels.length === 0" class="panel">
          <p class="empty">{{ zh.novelsEmpty }}</p>
        </section>

        <template v-else>
        <section class="export-layout">
          <section class="panel export-sidebar task-grid">
            <div class="summary-grid span-2">
              <article v-for="item in exportSummary" :key="item.label" class="summary-card">
                <span>{{ item.label }}</span>
                <strong>{{ item.value }}</strong>
              </article>
            </div>

            <label class="field span-2">
              <span>{{ zh.novel }}</span>
              <select v-model="exportForm.novelId" @change="handleNovelChange">
                <option v-for="novel in state.novels" :key="novel.id" :value="novel.id">
                  {{ novel.title }}
                </option>
              </select>
            </label>

            <label class="field">
              <span>{{ zh.chapterLimit }}</span>
              <input v-model="exportForm.maxChapters" type="number" min="0" :placeholder="zh.chapterLimitPlaceholder" />
            </label>

            <label class="field">
              <span>{{ zh.retryCount }}</span>
              <input v-model="exportForm.retryCount" type="number" min="0" :placeholder="zh.retryPlaceholder" />
            </label>

            <label class="toggle span-2">
              <input v-model="exportForm.skipFilteredTitle" type="checkbox" />
              <span>{{ zh.skipFiltered }}</span>
            </label>

            <label class="toggle span-2">
              <input v-model="exportForm.skipOnFailure" type="checkbox" />
              <span>{{ zh.skipOnFailure }}</span>
            </label>

            <div class="actions span-2">
              <button type="button" class="primary" :disabled="state.analyzing || state.loading" @click="analyzeCatalog">
                {{ state.analyzing ? zh.parsing : zh.parseCatalog }}
              </button>
              <button type="button" class="accent" :disabled="state.exporting || state.loading" @click="exportNovel">
                {{ state.exporting ? zh.exporting : zh.exportTxt }}
              </button>
            </div>

            <section class="panel export-controls span-2">
              <div class="export-control-block">
                <div class="panel-head compact-head">
                  <h3>{{ zh.rangeInput }}</h3>
                </div>
                <label class="field">
                  <input
                    v-model="exportForm.rangeInput"
                    :placeholder="zh.rangePlaceholder"
                    @change="commitRangeSelection"
                    @blur="commitRangeSelection"
                    @keyup.enter.prevent="commitRangeSelection"
                  />
                </label>
              </div>

              <div class="export-control-block">
                <div class="chapter-tools">
                  <div class="tool-row">
                    <label class="field">
                      <span>{{ zh.keywordInput }}</span>
                      <input
                        v-model="exportForm.keywordInput"
                        :placeholder="zh.keywordPlaceholder"
                        @change="commitKeywordSelection"
                        @blur="commitKeywordSelection"
                        @keyup.enter.prevent="commitKeywordSelection"
                      />
                    </label>
                  </div>

                  <div class="tool-row wrap">
                    <label class="toggle">
                      <input v-model="exportForm.showMatchedOnly" type="checkbox" />
                      <span>{{ zh.showMatchedOnly }}</span>
                    </label>
                    <label class="toggle">
                      <input v-model="exportForm.showSelectedOnly" type="checkbox" />
                      <span>{{ zh.showSelectedOnly }}</span>
                    </label>
                    <button type="button" class="ghost small-btn" @click="selectMatchedOnCurrentPage">{{ zh.selectMatchedOnPage }}</button>
                    <button type="button" class="ghost small-btn" @click="clearMatchedOnCurrentPage">{{ zh.clearMatchedOnPage }}</button>
                    <button type="button" class="ghost small-btn" @click="selectAllChapters">{{ zh.selectAll }}</button>
                    <button type="button" class="ghost small-btn" @click="clearChapterSelection">{{ zh.clearSelection }}</button>
                    <button type="button" class="ghost small-btn" @click="invertChapterSelection">{{ zh.invertSelection }}</button>
                    <button type="button" class="ghost small-btn" @click="toggleCurrentPageSelection">{{ zh.currentPageToggle }}</button>
                  </div>
                </div>
              </div>
            </section>
          </section>

          <section class="panel preview export-main">
            <div class="panel-head">
              <h2>{{ zh.preview }}</h2>
              <span v-if="state.analysis">{{ state.analysis.chapterCount }} {{ zh.chapters }}</span>
            </div>

            <template v-if="state.analysis">
              <div class="preview-head compact">
                <p class="preview-summary">
                  <strong>{{ zh.selectedChapters }}：{{ selectedChapterCount }} / {{ analyzedChapters.length }}</strong>
                  <span>{{ zh.pageStatus }} {{ exportForm.currentPage }} / {{ totalPages }}</span>
                  <span class="page-nav-inline">
                    <button type="button" class="ghost small-btn" :disabled="exportForm.currentPage <= 1" @click="setPage(exportForm.currentPage - 1)">
                      {{ zh.previousPage }}
                    </button>
                    <button type="button" class="ghost small-btn" :disabled="exportForm.currentPage >= totalPages" @click="setPage(exportForm.currentPage + 1)">
                      {{ zh.nextPage }}
                    </button>
                  </span>
                </p>
              </div>

              <div class="chapter-list">
                <article
                  v-for="(chapter, index) in pagedChapters"
                  :key="chapter.url"
                  class="chapter-card selectable"
                  :class="{ matched: matchedChapterUrls.has(chapter.url) }"
                >
                  <label class="chapter-check">
                    <input
                      type="checkbox"
                      :checked="selectedChapterSet.has(chapter.url)"
                      @change="toggleChapterSelection(chapter.url, $event.target.checked)"
                    />
                  <strong class="chapter-title">
                    {{ (exportForm.currentPage - 1) * exportForm.pageSize + index + 1 }}. {{ chapter.title }}
                    <small v-if="chapter.cached" class="match-badge cache-badge">{{ zh.cachedBadge }}</small>
                    <small v-if="matchedChapterUrls.has(chapter.url)" class="match-badge">{{ zh.matchedBadge }}</small>
                  </strong>
                </label>
                  <small class="chapter-url">{{ chapter.url }}</small>
                </article>
              </div>

            </template>

            <p v-else class="empty">{{ zh.previewEmpty }}</p>
          </section>
        </section>
        </template>
      </template>

      <template v-else-if="uiState.activeTab === 'novels'">
        <section class="two-col">
          <div class="panel scroll-panel">
            <div class="panel-head">
              <h2>{{ zh.novels }}</h2>
              <button class="ghost" type="button" @click="resetNovelForm">{{ zh.newNovel }}</button>
            </div>
            <div class="panel-scroll-body">
              <div class="form-grid compact-grid" style="margin-bottom: 12px;">
                <label class="field span-2">
                  <span>{{ zh.novelSearch }}</span>
                  <input v-model="listFilters.search" :placeholder="zh.novelSearchPlaceholder" />
                </label>
                <label class="field span-2">
                  <span>{{ zh.filterRule }}</span>
                  <select v-model="listFilters.ruleId">
                    <option value="">{{ zh.allRules }}</option>
                    <option v-for="rule in state.rules" :key="rule.id" :value="rule.id">
                      {{ rule.name }}
                    </option>
                  </select>
                </label>
              </div>

              <div class="rule-list">
                <button
                  v-for="novel in filteredNovels"
                  :key="novel.id"
                  class="rule-chip"
                  :class="{ active: exportForm.novelId === novel.id }"
                  type="button"
                  @click="pickNovel(novel)"
                >
                  <span>{{ novel.title }}</span>
                  <small>{{ novel.catalogUrl }}</small>
                </button>
              </div>
              <p v-if="state.novels.length === 0" class="empty">{{ zh.novelsEmpty }}</p>
              <p v-else-if="filteredNovels.length === 0" class="empty">{{ zh.novelsFilteredEmpty }}</p>
            </div>
          </div>

          <div class="panel scroll-panel">
            <div class="panel-head">
              <h2>{{ zh.novelEditor }}</h2>
              <span v-if="selectedNovel">{{ zh.current }}: {{ selectedNovel.title }}</span>
            </div>
            <div class="panel-scroll-body">
              <div class="form-grid">
                <label class="field">
                  <span>{{ zh.novelId }}</span>
                  <input v-model="novelForm.id" :placeholder="zh.novelIdPlaceholder" />
                </label>

                <label class="field">
                  <span>{{ zh.boundRule }}</span>
                  <select v-model="novelForm.ruleId" @change="handleNovelRuleChange">
                    <option v-for="rule in state.rules" :key="rule.id" :value="rule.id">
                      {{ rule.name }}
                    </option>
                  </select>
                </label>

                <label class="field span-2">
                  <span>{{ zh.novelTitle }}</span>
                  <input v-model="novelForm.title" :placeholder="zh.novelTitlePlaceholder" />
                </label>

                <label class="field span-2">
                  <span>{{ zh.catalogUrl }}</span>
                  <input v-model="novelForm.catalogUrl" :placeholder="zh.catalogPlaceholder" />
                </label>
              </div>

              <div class="actions sticky-actions">
                <button type="button" class="primary" :disabled="state.savingNovel" @click="saveNovel">
                  {{ state.savingNovel ? zh.savingNovel : zh.saveNovel }}
                </button>
                <button type="button" class="ghost danger" :disabled="!novelForm.id" @click="removeNovel">
                  {{ zh.deleteNovel }}
                </button>
              </div>
            </div>
          </div>
        </section>
      </template>

      <template v-else-if="uiState.activeTab === 'rules'">
        <section class="two-col">
          <div class="panel scroll-panel">
            <div class="panel-head">
              <h2>{{ zh.rules }}</h2>
              <button class="ghost" type="button" @click="resetRuleForm">{{ zh.newRule }}</button>
            </div>
            <div class="panel-scroll-body">
              <div class="rule-list">
                <button
                  v-for="rule in state.rules"
                  :key="rule.id"
                  class="rule-chip"
                  :class="{ active: selectedRule?.id === rule.id }"
                  type="button"
                  @click="pickRule(rule)"
                >
                  <span>{{ rule.name }}</span>
                  <small>{{ (rule.matchDomains ?? []).join(', ') }}</small>
                </button>
              </div>
              <p v-if="state.rules.length === 0" class="empty">{{ zh.rulesEmpty }}</p>
            </div>
          </div>

          <div class="panel scroll-panel">
            <div class="panel-head">
              <h2>{{ zh.ruleEditor }}</h2>
              <span v-if="selectedRule">{{ zh.current }}: {{ selectedRule.name }}</span>
            </div>
            <div class="panel-scroll-body">
            <div class="form-grid">
              <label class="field">
                <span>{{ zh.ruleId }}</span>
                <input v-model="ruleForm.id" :placeholder="zh.ruleIdPlaceholder" />
              </label>

              <label class="field">
                <span>{{ zh.ruleName }}</span>
                <input v-model="ruleForm.name" :placeholder="zh.ruleNamePlaceholder" />
              </label>

              <label class="field span-2">
                <span class="field-label">{{ zh.matchDomains }} <button type="button" class="help-toggle" @click.prevent="toggleHelp('matchDomains')">?</button></span>
                <textarea v-model="ruleForm.matchDomains" rows="3" :placeholder="zh.matchDomainsPlaceholder"></textarea>
                <small v-if="uiState.helpOpen.matchDomains" class="field-help">{{ zh.matchDomainsHelp }}</small>
              </label>

              <label class="field">
                <span class="field-label">{{ zh.catalogSelector }} <button type="button" class="help-toggle" @click.prevent="toggleHelp('catalogSelector')">?</button></span>
                <input v-model="ruleForm.catalogChapterLinkSelector" :placeholder="zh.catalogSelectorPlaceholder" />
                <small v-if="uiState.helpOpen.catalogSelector" class="field-help">{{ zh.catalogSelectorHelp }}</small>
              </label>

              <label class="field">
                <span class="field-label">{{ zh.contentSelector }} <button type="button" class="help-toggle" @click.prevent="toggleHelp('contentSelector')">?</button></span>
                <input v-model="ruleForm.chapterContentSelector" :placeholder="zh.contentSelectorPlaceholder" />
                <small v-if="uiState.helpOpen.contentSelector" class="field-help">{{ zh.contentSelectorHelp }}</small>
              </label>

              <label class="field">
                <span class="field-label">{{ zh.headingText }} <button type="button" class="help-toggle" @click.prevent="toggleHelp('headingText')">?</button></span>
                <input v-model="ruleForm.catalogSectionHeadingText" :placeholder="zh.headingPlaceholder" />
                <small v-if="uiState.helpOpen.headingText" class="field-help">{{ zh.headingHelp }}</small>
              </label>

              <label class="field">
                <span class="field-label">{{ zh.sectionSelector }} <button type="button" class="help-toggle" @click.prevent="toggleHelp('sectionSelector')">?</button></span>
                <input v-model="ruleForm.catalogSectionContainer" :placeholder="zh.sectionPlaceholder" />
                <small v-if="uiState.helpOpen.sectionSelector" class="field-help">{{ zh.sectionHelp }}</small>
              </label>

              <label class="field">
                <span class="field-label">{{ zh.nextPageSelector }} <button type="button" class="help-toggle" @click.prevent="toggleHelp('nextPageSelector')">?</button></span>
                <input v-model="ruleForm.nextPageSelector" :placeholder="zh.nextPagePlaceholder" />
                <small v-if="uiState.helpOpen.nextPageSelector" class="field-help">{{ zh.nextPageHelp }}</small>
              </label>

              <label class="field">
                <span class="field-label">{{ zh.nextChapterSelector }} <button type="button" class="help-toggle" @click.prevent="toggleHelp('nextChapterSelector')">?</button></span>
                <input v-model="ruleForm.nextChapterSelector" :placeholder="zh.nextChapterPlaceholder" />
                <small v-if="uiState.helpOpen.nextChapterSelector" class="field-help">{{ zh.nextChapterHelp }}</small>
              </label>

              <label class="field span-2">
                <span class="field-label">{{ zh.cleanupSelectors }} <button type="button" class="help-toggle" @click.prevent="toggleHelp('cleanupSelectors')">?</button></span>
                <textarea v-model="ruleForm.contentCleanupSelectors" rows="3" :placeholder="zh.cleanupSelectorsPlaceholder"></textarea>
                <small v-if="uiState.helpOpen.cleanupSelectors" class="field-help">{{ zh.cleanupSelectorsHelp }}</small>
              </label>

              <label class="field span-2">
                <span class="field-label">{{ zh.stopTexts }} <button type="button" class="help-toggle" @click.prevent="toggleHelp('stopTexts')">?</button></span>
                <textarea v-model="ruleForm.contentStopTexts" rows="3" :placeholder="zh.stopTextsPlaceholder"></textarea>
                <small v-if="uiState.helpOpen.stopTexts" class="field-help">{{ zh.stopTextsHelp }}</small>
              </label>

              <label class="field span-2">
                <span class="field-label">{{ zh.removeMatchingLines }} <button type="button" class="help-toggle" @click.prevent="toggleHelp('removeMatchingLines')">?</button></span>
                <textarea v-model="ruleForm.removeMatchingLines" rows="3" :placeholder="zh.removeMatchingLinesPlaceholder"></textarea>
                <small v-if="uiState.helpOpen.removeMatchingLines" class="field-help">{{ zh.removeMatchingLinesHelp }}</small>
              </label>

              <div class="field span-2">
                <span class="field-label">{{ zh.textReplaceRules }} <button type="button" class="help-toggle" @click.prevent="toggleHelp('textReplaceRules')">?</button></span>
                <small v-if="uiState.helpOpen.textReplaceRules" class="field-help">{{ zh.textReplaceRulesHelp }}</small>
                <div class="rule-group-list">
                  <div v-for="(item, index) in ruleForm.textReplacementRules" :key="`text-${index}`" class="subrule-card">
                    <div class="subrule-head">
                      <strong>{{ zh.textReplaceRules }} {{ index + 1 }}</strong>
                      <button type="button" class="ghost danger small-btn" @click="removeTextReplacementRule(index)">删除</button>
                    </div>
                    <div class="subrule-fields">
                      <label class="field">
                        <span>{{ zh.matchText }}</span>
                        <input v-model="item.match" :placeholder="zh.matchText" />
                      </label>
                      <label class="field">
                        <span>{{ zh.replaceText }}</span>
                        <input v-model="item.replace" :placeholder="zh.replacePlaceholder" />
                      </label>
                    </div>
                    <div class="subrule-options">
                      <label class="toggle">
                        <input v-model="item.enabled" type="checkbox" />
                        <span>{{ zh.ruleEnabled }}</span>
                      </label>
                      <label class="toggle">
                        <input v-model="item.caseSensitive" type="checkbox" />
                        <span>{{ zh.caseSensitive }}</span>
                      </label>
                      <label class="toggle">
                        <input v-model="item.replaceFirst" type="checkbox" />
                        <span>{{ zh.replaceFirst }}</span>
                      </label>
                    </div>
                  </div>
                </div>
                <div class="actions compact">
                  <button type="button" class="ghost" @click="addTextReplacementRule">{{ zh.addRule }}</button>
                </div>
              </div>

              <div class="field span-2">
                <span class="field-label">{{ zh.regexRules }} <button type="button" class="help-toggle" @click.prevent="toggleHelp('regexRules')">?</button></span>
                <small v-if="uiState.helpOpen.regexRules" class="field-help">{{ zh.regexRulesHelp }}</small>
                <div class="rule-group-list">
                  <div v-for="(item, index) in ruleForm.regexReplacementRules" :key="`regex-${index}`" class="subrule-card">
                    <div class="subrule-head">
                      <strong>{{ zh.regexRules }} {{ index + 1 }}</strong>
                      <button type="button" class="ghost danger small-btn" @click="removeRegexReplacementRule(index)">删除</button>
                    </div>
                    <div class="subrule-fields">
                      <label class="field">
                        <span>{{ zh.regexPattern }}</span>
                        <input v-model="item.pattern" :placeholder="zh.regexPlaceholder" />
                      </label>
                      <label class="field">
                        <span>{{ zh.replaceText }}</span>
                        <input v-model="item.replace" :placeholder="zh.replacePlaceholder" :disabled="item.removeLine" />
                      </label>
                    </div>
                    <div class="subrule-options">
                      <label class="toggle">
                        <input v-model="item.enabled" type="checkbox" />
                        <span>{{ zh.ruleEnabled }}</span>
                      </label>
                      <label class="toggle">
                        <input v-model="item.removeLine" type="checkbox" />
                        <span>{{ zh.removeLine }}</span>
                      </label>
                      <label class="toggle">
                        <input v-model="item.replaceFirst" type="checkbox" :disabled="item.removeLine" />
                        <span>{{ zh.replaceFirst }}</span>
                      </label>
                    </div>
                  </div>
                </div>
                <div class="actions compact">
                  <button type="button" class="ghost" @click="addRegexReplacementRule">{{ zh.addRule }}</button>
                </div>
              </div>

              <label class="field span-2">
                <span class="field-label">{{ zh.skipRegex }} <button type="button" class="help-toggle" @click.prevent="toggleHelp('skipRegex')">?</button></span>
                <textarea v-model="ruleForm.skipChapterTitlePatterns" rows="4" :placeholder="zh.skipRegexPlaceholder"></textarea>
                <small v-if="uiState.helpOpen.skipRegex" class="field-help">{{ zh.skipRegexHelp }}</small>
              </label>

              <label class="field span-2">
                <span>{{ zh.notes }}</span>
                <textarea v-model="ruleForm.notes" rows="3" :placeholder="zh.notesPlaceholder"></textarea>
              </label>
            </div>

            <div class="actions sticky-actions">
              <button type="button" class="primary" :disabled="state.savingRule" @click="saveRule">
                {{ state.savingRule ? zh.savingRule : zh.saveRule }}
              </button>
              <button type="button" class="ghost danger" :disabled="!ruleForm.id" @click="removeRule">
                {{ zh.deleteRule }}
              </button>
            </div>
            </div>
          </div>
        </section>
      </template>

      <template v-else-if="uiState.activeTab === 'cache'">
        <section class="panel">
          <div class="panel-head">
            <h2>{{ zh.cache }}</h2>
            <button class="ghost" type="button" :disabled="state.clearingCache || cacheForm.selectedNovelIds.length === 0" @click="clearSelectedCaches">
              {{ state.clearingCache ? zh.clearingCache : zh.clearCache }}
            </button>
          </div>

          <div v-if="state.caches.length > 0" class="cache-list">
            <label v-for="cache in state.caches" :key="cache.novelId" class="cache-item">
              <input v-model="cacheForm.selectedNovelIds" type="checkbox" :value="cache.novelId" />
              <div>
                <strong>{{ cache.novelTitle }}</strong>
                <small>{{ cache.ruleName }}</small>
                <small>{{ zh.cacheFiles }}: {{ cache.fileCount }} | {{ zh.cacheSize }}: {{ formatBytes(cache.totalBytes) }}</small>
              </div>
            </label>
          </div>
          <p v-else class="empty">{{ zh.cacheEmpty }}</p>
        </section>
      </template>
      </section>
    </main>
  </div>
</template>
