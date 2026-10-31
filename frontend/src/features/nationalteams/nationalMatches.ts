import type { NationalMatch } from '../../api/types'
import { COUNTRY_NAME_KEYS } from '../../api/types'
import { regionForCountry, type Region } from '../../components/avatar/traits'
import { intlLocale } from '../../utils/locale'
import { roundName } from '../cup/CupPage'

type TFunction = (key: string, options?: Record<string, unknown>) => string

export const NATIONAL_TEAMS_TABS = ['round', 'results', 'world-cup'] as const
export type NationalTeamsTab = (typeof NATIONAL_TEAMS_TABS)[number]

// ?tab= picks the national-teams page tab, so a link can open straight onto the World Cup;
// anything else falls back to the current round.
export function nationalTeamsTabFromParam(param: string | null): NationalTeamsTab {
  return NATIONAL_TEAMS_TABS.find((tab) => tab === param) ?? 'round'
}

export const NATIONAL_REGIONS: Region[] = ['europe', 'africa', 'asia', 'americas']

// Where a national-team row leads: a weekly friendly or a World Cup group match is a friendly
// under the hood (/friendlies/:id), a knockout tie a cup tie (/cup/ties/:id).
export function nationalMatchLink(match: NationalMatch): string {
  return match.kind === 'worldCupKnockout' ? `/cup/ties/${match.id}` : `/friendlies/${match.id}`
}

// The competition label on a row's first line: "Amistoso", "Mundial · Grupo C · J2" or
// "Mundial · Semifinal".
export function nationalMatchLabel(t: TFunction, match: NationalMatch): string {
  switch (match.kind) {
    case 'worldCupGroup':
      return t('nationalTeams.kind.worldCupGroup', { group: match.groupName ?? '', matchday: match.round ?? 1 })
    case 'worldCupKnockout':
      return t('nationalTeams.kind.worldCupKnockout', { round: roundName(t, match.round ?? 1, match.totalRounds ?? 0) })
    default:
      return t('nationalTeams.kind.friendly')
  }
}

export interface NationalMatchFilter {
  query: string
  region: Region | null
  onlyMine: boolean
}

export const EMPTY_FILTER: NationalMatchFilter = { query: '', region: null, onlyMine: false }

// Lower-cased, accent-free - so "espana" finds "España" and "COSTA" finds "Costa de Marfil".
function fold(text: string): string {
  return text
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .toLowerCase()
}

function countryName(t: TFunction, code: string): string {
  const key = COUNTRY_NAME_KEYS[code]
  return key ? t(key) : code
}

// Keeps a match when either side matches the search (its localized country name) and the region,
// and, with onlyMine, when the viewer has a player in its saved call-up.
export function matchesFilter(t: TFunction, match: NationalMatch, filter: NationalMatchFilter): boolean {
  if (filter.onlyMine && !(match.myPlayers && match.myPlayers > 0)) return false
  const sides = [match.homeCountry, match.awayCountry]
  if (filter.region && !sides.some((code) => regionForCountry(code) === filter.region)) return false
  const query = fold(filter.query.trim())
  if (query && !sides.some((code) => fold(countryName(t, code)).includes(query) || code === query)) return false
  return true
}

// The country codes whose localized name contains query, for the results tab's server-side
// country filter - only an exact single match is sent to the server.
export function countriesMatching(t: TFunction, query: string): string[] {
  const q = fold(query.trim())
  if (!q) return []
  const codes = Object.keys(COUNTRY_NAME_KEYS)
  const exact = codes.filter((code) => fold(countryName(t, code)) === q)
  if (exact.length > 0) return exact
  return codes.filter((code) => fold(countryName(t, code)).includes(q))
}

export interface KickoffGroup {
  key: string
  kickoff: string
  matches: NationalMatch[]
}

export interface RoundSections {
  live: NationalMatch[]
  upcoming: KickoffGroup[]
  finished: KickoffGroup[]
}

function groupByKickoff(matches: NationalMatch[]): KickoffGroup[] {
  const groups = new Map<string, KickoffGroup>()
  for (const match of matches) {
    const key = new Date(match.kickoff).toISOString()
    const group = groups.get(key) ?? { key, kickoff: match.kickoff, matches: [] }
    group.matches.push(match)
    groups.set(key, group)
  }
  return Array.from(groups.values())
}

// Splits the round into what is live now (soonest kickoff first), what is still to come (grouped
// by kickoff instant, soonest first) and what finished today (grouped by kickoff, latest first).
export function splitRound(matches: NationalMatch[]): RoundSections {
  const byKickoff = (a: NationalMatch, b: NationalMatch) => new Date(a.kickoff).getTime() - new Date(b.kickoff).getTime()
  const live = matches.filter((m) => m.status === 'live').sort(byKickoff)
  const upcoming = groupByKickoff(matches.filter((m) => m.status === 'scheduled').sort(byKickoff))
  const finished = groupByKickoff(matches.filter((m) => m.status === 'finished').sort((a, b) => byKickoff(b, a)))
  return { live, upcoming, finished }
}

// "sáb, 7 nov, 21:00" in the viewer's own timezone - a kickoff group's header.
export function formatKickoffHeader(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  return new Intl.DateTimeFormat(intlLocale(), {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(date)
}

export type FormLetter = 'W' | 'D' | 'L'

// teamId's results from newest to oldest as W/D/L, at most limit of them; matches without a final
// score are skipped.
export function formStrip(matches: NationalMatch[], teamId: number, limit = 5): FormLetter[] {
  const letters: FormLetter[] = []
  for (const match of matches) {
    if (letters.length >= limit) break
    if (match.status !== 'finished' || match.homeGoals === undefined || match.awayGoals === undefined) continue
    const isHome = match.homeTeamId === teamId
    if (!isHome && match.awayTeamId !== teamId) continue
    const own = isHome ? match.homeGoals : match.awayGoals
    const other = isHome ? match.awayGoals : match.homeGoals
    letters.push(own > other ? 'W' : own === other ? 'D' : 'L')
  }
  return letters
}
