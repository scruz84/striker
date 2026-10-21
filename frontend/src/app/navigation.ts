import { matchPath } from 'react-router-dom'

/** Pages that are a drill-down from somewhere else (a match, a player, another club, a cup tie…):
 * on a phone they get a back bar instead of the club subheader, since an installed PWA has no
 * browser back button. `fallback` is where "back" goes when the page was opened directly. */
export interface DetailRoute {
  pattern: string
  fallback: string
  labelKey: string
}

export const DETAIL_ROUTES: DetailRoute[] = [
  { pattern: '/match/:id', fallback: '/dashboard', labelKey: 'nav.detail.match' },
  { pattern: '/player/:id', fallback: '/dashboard', labelKey: 'nav.detail.player' },
  { pattern: '/cup/ties/:id', fallback: '/cup', labelKey: 'nav.detail.cupTie' },
  { pattern: '/friendlies/:id', fallback: '/friendlies', labelKey: 'nav.detail.friendly' },
  { pattern: '/division/:id', fallback: '/leagues', labelKey: 'nav.detail.division' },
  { pattern: '/players/evolutions', fallback: '/dashboard', labelKey: 'nav.detail.evolutions' },
  { pattern: '/team/:id', fallback: '/dashboard', labelKey: 'nav.detail.team' },
  { pattern: '/team/:id/matches', fallback: '/dashboard', labelKey: 'nav.detail.teamMatches' },
]

/** The detail route matching `pathname`, ignoring the user's own team pages (their squad and
 * matches are top-level "my club" destinations, not drill-downs). */
export function findDetailRoute(pathname: string, ownTeamId?: number): DetailRoute | undefined {
  const route = DETAIL_ROUTES.find((r) => matchPath(r.pattern, pathname))
  if (!route) {
    return undefined
  }
  if (route.pattern.startsWith('/team/') && ownTeamId !== undefined) {
    const params = matchPath(route.pattern, pathname)?.params
    if (Number(params?.id) === ownTeamId) {
      return undefined
    }
  }
  return route
}

/** Pages that are about the user's own club - the only ones that show the club subheader. */
export function isClubPage(pathname: string, ownTeamId?: number): boolean {
  if (['/dashboard', '/dashboard/lineup', '/dashboard/finances', '/transfers'].includes(pathname)) {
    return true
  }
  if (ownTeamId === undefined) {
    return false
  }
  return pathname === `/team/${ownTeamId}` || pathname === `/team/${ownTeamId}/matches`
}

/** Whether a nav entry should look selected on `pathname` - sub-routes count (e.g. a cup tie
 * keeps "Cup" selected), so people can tell which section they are in. */
export function isNavSelected(itemPath: string, pathname: string, alsoMatches: string[] = []): boolean {
  if (pathname === itemPath) {
    return true
  }
  return alsoMatches.some((pattern) => matchPath({ path: pattern, end: false }, pathname))
}

/** Roman tier numeral ("I", "III") to a number - division names look like "I.1" / "III.16". */
function romanToInt(roman: string): number {
  const values: Record<string, number> = { I: 1, V: 5, X: 10 }
  let total = 0
  for (let i = 0; i < roman.length; i++) {
    const current = values[roman[i]]
    const next = values[roman[i + 1]]
    total += next && next > current ? -current : current
  }
  return total
}

export interface ParsedDivisionName {
  tier: number
  group?: number
}

/** Splits a division code like "I.1" or "III.16" into tier and group; undefined for any other
 * (custom) division name, which is then shown verbatim. */
export function parseDivisionName(name: string): ParsedDivisionName | undefined {
  const match = /^([IVX]+)(?:\.(\d+))?$/.exec(name.trim())
  if (!match) {
    return undefined
  }
  return { tier: romanToInt(match[1]), group: match[2] ? Number(match[2]) : undefined }
}
