import assert from 'node:assert/strict'

import { normalizeWorkspaceTab, workspaceTabs } from './workspaceTabs.js'

assert.deepEqual(workspaceTabs, ['export', 'novels', 'rules', 'cache'])
assert.equal(normalizeWorkspaceTab('rules'), 'rules')
assert.equal(normalizeWorkspaceTab('unknown'), 'export')
