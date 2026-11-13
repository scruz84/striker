import { describe, expect, it } from 'vitest'
import en from '../i18n/locales/en.json'
import es from '../i18n/locales/es.json'
import { GOAL_SITUATIONS } from './matchCommentary'

// openspec match-commentary "Phrasing pool sizes": every pool, in both languages, has at least its
// minimum number of lines, and each line only uses the placeholders its pool fills in.

const CHANCE = ['player', 'team', 'opponent', 'keeper']
const PLAYER = ['player', 'team', 'opponent']
const SIDES = ['team', 'opponent']
const SITUATION = ['player', 'team', 'opponent', 'keeper', 'score', 'goals']
const BOOKEND = ['home', 'away']
const RESULT = ['home', 'away', 'score']

const POOLS: Array<[path: string, minimum: number, placeholders: string[]]> = [
  ['commentary.goal', 12, CHANCE],
  ['commentary.noGoal', 12, CHANCE],
  ['commentary.penaltyGoal', 6, CHANCE],
  ['commentary.penaltyMiss', 6, CHANCE],
  ['commentary.freeKickGoal', 6, CHANCE],
  ['commentary.freeKickMiss', 6, CHANCE],
  ['commentary.yellowCard', 8, PLAYER],
  ['commentary.redCard', 6, PLAYER],
  ['commentary.secondYellow', 4, PLAYER],
  ['commentary.injury', 6, PLAYER],
  ['commentary.substitution', 8, ['team', 'opponent', 'inPlayer', 'outPlayer']],
  ['commentary.abandoned', 3, SIDES],
  ['commentary.defaultResult', 2, SIDES],
  ['commentary.mentalityChange.attacking', 4, SIDES],
  ['commentary.mentalityChange.balanced', 4, SIDES],
  ['commentary.mentalityChange.defensive', 4, SIDES],
  ...GOAL_SITUATIONS.map((situation): [string, number, string[]] => [`commentary.situation.${situation}`, 4, SITUATION]),
  ['commentary.lateMiss', 4, PLAYER],
  ['commentary.colour.miss', 10, PLAYER],
  ['commentary.colour.yellowCard', 8, PLAYER],
  ['report.kickoff', 6, BOOKEND],
  ['report.halfTime.goalless', 3, RESULT],
  ['report.halfTime.level', 3, RESULT],
  ['report.halfTime.homeLeads', 3, RESULT],
  ['report.halfTime.awayLeads', 3, RESULT],
  ['report.fullTimeResult.goallessDraw', 3, RESULT],
  ['report.fullTimeResult.scoringDraw', 3, RESULT],
  ['report.fullTimeResult.homeWin', 3, RESULT],
  ['report.fullTimeResult.awayWin', 3, RESULT],
  ['report.fullTimeResult.bigWin', 3, [...RESULT, 'winner', 'loser']],
]

const LOCALES = { en, es } as const

function poolAt(locale: unknown, path: string): unknown {
  return path.split('.').reduce<unknown>((node, part) => (node as Record<string, unknown> | undefined)?.[part], (locale as { match: unknown }).match)
}

function placeholdersOf(line: string): string[] {
  return [...line.matchAll(/\{\{(\w+)\}\}/g)].map((match) => match[1])
}

/** Returns the problems with one pool in one locale - empty when the pool is fine. */
function poolProblems(lang: string, path: string, pool: unknown, minimum: number, allowed: string[]): string[] {
  if (!Array.isArray(pool)) return [`${lang} ${path}: missing`]
  const problems: string[] = []
  if (pool.length < minimum) problems.push(`${lang} ${path}: ${pool.length} lines, needs ${minimum}`)
  for (const line of pool as string[]) {
    for (const name of placeholdersOf(line)) {
      if (!allowed.includes(name)) problems.push(`${lang} ${path}: unknown placeholder {{${name}}} in "${line}"`)
    }
    // An event stored before the keeper was recorded fills {{keeper}} with "the goalkeeper" /
    // "el portero", so it must never start a sentence (it would read lower-case) nor follow the
    // Spanish "a" / "de" (which would read "a el" / "de el" instead of "al" / "del").
    if (/(^|[.!?¡¿]\s*)\{\{keeper\}\}/.test(line)) problems.push(`${lang} ${path}: {{keeper}} starts a sentence in "${line}"`)
    if (lang === 'es' && /\b(a|de)\s+\{\{keeper\}\}/i.test(line)) problems.push(`${lang} ${path}: "a/de {{keeper}}" in "${line}"`)
    // A substitution is shown live while injuries are hidden, so it must never say why the
    // player came off (spec "Commentary stays the same from live to finished").
    if (path === 'commentary.substitution' && /lesi|injur|cojea|limp/i.test(line)) problems.push(`${lang} ${path}: mentions an injury in "${line}"`)
  }
  return problems
}

describe('match commentary locale pools', () => {
  for (const [lang, locale] of Object.entries(LOCALES)) {
    it(`meets every pool minimum and placeholder rule in ${lang}`, () => {
      const problems = POOLS.flatMap(([path, minimum, allowed]) => poolProblems(lang, path, poolAt(locale, path), minimum, allowed))
      expect(problems).toEqual([])
    })
  }

  it('keeps the neutral full-time line for abandoned and default-result matches', () => {
    expect(typeof en.match.report.fullTime).toBe('string')
    expect(typeof es.match.report.fullTime).toBe('string')
  })

  it('catches a short pool, an unknown placeholder and a bad keeper position', () => {
    expect(poolProblems('en', 'commentary.goal', ['GOAL!'], 2, CHANCE)).toEqual(['en commentary.goal: 1 lines, needs 2'])
    expect(poolProblems('en', 'commentary.goal', ['{{striker}} scores'], 1, CHANCE)).toHaveLength(1)
    expect(poolProblems('en', 'commentary.noGoal', ['Saved! {{keeper}} gathers.'], 1, CHANCE)).toHaveLength(1)
    expect(poolProblems('es', 'commentary.noGoal', ['{{player}} bate a {{keeper}}.'], 1, CHANCE)).toHaveLength(1)
    expect(poolProblems('es', 'commentary.substitution', ['{{outPlayer}} se retira lesionado.'], 1, ['outPlayer'])).toHaveLength(1)
  })
})
