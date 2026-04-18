import assert from 'node:assert/strict'

import { filterNovels } from './novelListFilters.js'

const novels = [
  { id: '1', title: '青山', ruleId: 'zhswx' },
  { id: '2', title: '凡人修仙传', ruleId: 'zhswx' },
  { id: '3', title: '诡秘之主', ruleId: '23shuku' },
]

assert.deepEqual(filterNovels(novels, '青', '').map((item) => item.id), ['1'])
assert.deepEqual(filterNovels(novels, '', '23shuku').map((item) => item.id), ['3'])
assert.deepEqual(filterNovels(novels, '之', '23shuku').map((item) => item.id), ['3'])
