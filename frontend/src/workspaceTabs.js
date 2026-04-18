export const workspaceTabs = ['export', 'novels', 'rules', 'cache']

export function normalizeWorkspaceTab(tab) {
  return workspaceTabs.includes(tab) ? tab : 'export'
}
