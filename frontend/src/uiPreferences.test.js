import assert from 'node:assert/strict'

import { getSavedWorkspaceTab } from './uiPreferences.js'

assert.equal(getSavedWorkspaceTab({ getItem: () => 'rules' }), 'rules')
assert.equal(getSavedWorkspaceTab({ getItem: () => null }), 'export')
assert.equal(getSavedWorkspaceTab({ getItem: () => 'bad-tab' }), 'export')
