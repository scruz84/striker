export const ADMIN_TABS = ['overview', 'teams', 'rules', 'world', 'users'] as const
export type AdminTab = (typeof ADMIN_TABS)[number]
export const DEFAULT_ADMIN_TAB: AdminTab = 'overview'

// adminTabFromParam maps the ?tab= query value onto a known tab, falling back to the default for a
// missing or unrecognised one (a stale or hand-edited link must never leave the page blank).
export function adminTabFromParam(param: string | null): AdminTab {
  return (ADMIN_TABS as readonly string[]).includes(param ?? '') ? (param as AdminTab) : DEFAULT_ADMIN_TAB
}
