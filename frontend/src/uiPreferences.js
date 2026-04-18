import { normalizeWorkspaceTab } from './workspaceTabs.js'

const ACTIVE_TAB_KEY = 'xiaoshuo.activeTab'

export function getSavedWorkspaceTab(storage = globalThis?.localStorage) {
  const rawValue = storage?.getItem?.(ACTIVE_TAB_KEY)
  return normalizeWorkspaceTab(rawValue)
}

export function saveWorkspaceTab(tab, storage = globalThis?.localStorage) {
  storage?.setItem?.(ACTIVE_TAB_KEY, normalizeWorkspaceTab(tab))
}
