import assert from 'node:assert/strict'

import { matchChapterByKeywords, parseKeywordInput } from './chapterFilters.js'

assert.deepEqual(parseKeywordInput(' 请假, 感言 ,,番外 '), ['请假', '感言', '番外'])
assert.equal(matchChapterByKeywords({ title: '第10章 番外篇' }, ['番外']), true)
assert.equal(matchChapterByKeywords({ title: '正文章节' }, ['番外']), false)
