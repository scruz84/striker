import type { ClassificationPosition, LeagueLiveScore, WeekMatch } from '../../api/types'
import { mightStillBeLive } from '../../api/useLiveMatch'

// The standings' own ordering (see the backend's GetDivisionDetailsStmtSql): points, goal
// difference, goals scored, wins, draws, then the season's starting order (no seed last) and the
// team id. Shared by hideStillLiveResults and the live table so both orderings stay identical.
export function compareClassification(a: ClassificationPosition, b: ClassificationPosition): number {
  if (b.points !== a.points) return b.points - a.points
  const diffA = a.goalsScored - a.goalsReceived
  const diffB = b.goalsScored - b.goalsReceived
  if (diffB !== diffA) return diffB - diffA
  if (b.goalsScored !== a.goalsScored) return b.goalsScored - a.goalsScored
  if (b.wins !== a.wins) return b.wins - a.wins
  if (b.draws !== a.draws) return b.draws - a.draws
  const seedA = a.seed ?? Number.MAX_SAFE_INTEGER
  const seedB = b.seed ?? Number.MAX_SAFE_INTEGER
  if (seedA !== seedB) return seedA - seedB
  return a.teamId - b.teamId
}

export interface LiveTable {
  positions: ClassificationPosition[]
  // Places gained (+) or lost (-) against the pre-matchday table, by team; a team that did not move
  // is absent.
  deltas: Map<number, number>
  // Teams whose match is on air right now (live or half-time), not yet over.
  playing: Set<number>
}

// applyLiveScores builds the live table from the standings with every still-live result reverted
// (hideStillLiveResults' output): for each reverted match whose broadcast entry is live, half-time
// or finished it adds one played match with the broadcast score, re-sorts with the standings' own
// ordering and renumbers. Only matches that hideStillLiveResults would revert are considered, so an
// entry of another week, or a pending one, adds nothing and a stray entry cannot double-count.
export function applyLiveScores(
  revertedPositions: ClassificationPosition[],
  previousWeekMatches: WeekMatch[],
  liveById: Map<number, LeagueLiveScore>,
): LiveTable {
  const entryByTeam = new Map<number, LeagueLiveScore>()
  for (const match of previousWeekMatches) {
    if (!match.played || !mightStillBeLive(match.matchDate)) continue
    const entry = liveById.get(match.matchId)
    if (!entry || entry.status === 'pending') continue
    entryByTeam.set(match.home, entry)
    entryByTeam.set(match.away, entry)
  }

  const before = new Map(revertedPositions.map((row) => [row.teamId, row.position]))
  const playing = new Set<number>()
  const updated = revertedPositions.map((row) => {
    const entry = entryByTeam.get(row.teamId)
    if (!entry) return row
    if (entry.status === 'live' || entry.status === 'halftime') playing.add(row.teamId)
    const isHome = entry.home === row.teamId
    const goalsFor = isHome ? entry.homeGoals : entry.awayGoals
    const goalsAgainst = isHome ? entry.awayGoals : entry.homeGoals
    const won = goalsFor > goalsAgainst
    const drawn = goalsFor === goalsAgainst
    return {
      ...row,
      points: row.points + (won ? 3 : drawn ? 1 : 0),
      matches: row.matches + 1,
      wins: row.wins + (won ? 1 : 0),
      draws: row.draws + (drawn ? 1 : 0),
      losses: row.losses + (!won && !drawn ? 1 : 0),
      goalsScored: row.goalsScored + goalsFor,
      goalsReceived: row.goalsReceived + goalsAgainst,
    }
  })

  updated.sort(compareClassification)
  const positions = updated.map((row, index) => ({ ...row, position: index + 1 }))
  const deltas = new Map<number, number>()
  for (const row of positions) {
    const delta = (before.get(row.teamId) ?? row.position) - row.position
    if (delta !== 0) deltas.set(row.teamId, delta)
  }
  return { positions, deltas, playing }
}
