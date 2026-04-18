import assert from 'node:assert/strict'

import { findChapterIndex, getChapterNeighbors, normalizeNovelWorkspaceView } from './novelReader.js'

const chapters = [
  { title: 'Chapter 1', url: 'u1' },
  { title: 'Chapter 2', url: 'u2' },
  { title: 'Chapter 3', url: 'u3' },
]

assert.equal(findChapterIndex(chapters, 'u2'), 1)
assert.deepEqual(getChapterNeighbors(chapters, 'u2'), {
  index: 1,
  previous: chapters[0],
  current: chapters[1],
  next: chapters[2],
})
assert.equal(normalizeNovelWorkspaceView('bad-value'), 'directory')
