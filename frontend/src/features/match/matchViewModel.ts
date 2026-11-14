import type { TFunction } from 'i18next'
import { queryKeys } from '../../api/hooks'
import {
  MATCH_KIND_CUP,
  MATCH_KIND_FRIENDLY,
  MATCH_KIND_LEAGUE,
  type CupDetails,
  type CupTieDetails,
  type DivisionDetails,
  type FriendlyMatch,
  type Lineup,
  type MatchEvent,
  type MatchInstant,
  type Player,
  type PlayerRating,
  type WeekMatch,
} from '../../api/types'
import { translateCupName } from '../../utils/ledgerDescription'
import { liveMatchPath } from '../../utils/matchSummary'
import { localizedTeamName } from '../../utils/nationalTeamName'
import { cupTiePenaltyWinnerName, expectedTotalRounds, isFirstRoundPlayIn, roundName } from '../cup/CupPage'
import { hideStillLiveResults } from '../division/DivisionDetailsPage'

// The one shape the shared match viewer (MatchViewer) renders, whatever the competition. Each of the
// three routes loads its own endpoint and maps the result onto this with a pure adapter below, so the
// viewer itself never asks "is this a league match, a cup tie or a friendly?" - the only places that
// differ are the adapters. Replaces the old cupTieToWeekMatch / friendlyToWeekMatch, which squeezed a
// cup tie or friendly into the league DTO (WeekMatch) with dummy zeros and drifted apart from it.
export interface MatchView {
  homeId: number
  homeName: string
  awayId: number
  awayName: string
  // The home team's stadium; empty when it has none (a national team), which hides the venue caption
  // and the stadium-entry banner.
  homeStadiumName: string
  // League standing chips (ClassificationPosition.position) - league only, since a position has no
  // meaning in a cup tie or friendly.
  homePosition?: number
  awayPosition?: number
  // Kickoff time, or '' when unknown.
  matchDate: string
  played: boolean
  homeGoals: number
  awayGoals: number
  events: MatchEvent[] | null
  matchInstants: MatchInstant[] | null
  homeLineup: Lineup | null
  awayLineup: Lineup | null
  homePlayers: Record<string, Player> | null
  awayPlayers: Record<string, Player> | null
  ratings?: Record<string, number>
  // Every player who featured, with his rating breakdown - see PlayerRating.
  playerRatings?: PlayerRating[]
  // The rivalry level (RIVALRY_*) - league matches and cup ties only.
  rivalry?: number
  // Report flavor (see backend common.UpdateMatchFlavor) - absent for a match played before it was stored.
  weather?: string
  attendance?: number
  refereeName?: string
  assistant1Name?: string
  assistant2Name?: string
  // "Jornada 3 de 38", "Copa del Rey · Cuartos de final", "Mundial · Grupo B · Jornada 2" or "Amistoso";
  // undefined while what it needs (the cup details) is still loading, in which case the subtitle falls
  // back to the date alone.
  subtitleLabel?: string
  // WebSocket path of the match's live broadcast.
  liveUrl: string
  // react-query key invalidated once the live broadcast completes, so the finished view reads the real data.
  queryKey: readonly unknown[]
  // Cup only: a tie level after 90 minutes shows a shootout line naming the winner.
  wentToPenalties?: boolean
  penaltyWinnerName?: string
}

// weeksInASeason derives how many matchdays a division's season runs for from its team count -
// a standard full-season double round robin (see backend engine.randomWeekMatches2, the only
// place this figure is actually computed server-side; there's no dedicated "total weeks" field
// to fetch instead, so this mirrors that same formula from the team count the matchday label
// already needs division details for anyway).
export function weeksInASeason(teamCount: number): number {
  return (teamCount - 1) * 2
}

// A team's current standing in its league (see ClassificationPosition.position) - undefined
// while the division hasn't loaded yet, or if that team isn't in it (shouldn't happen for a
// league match's own two teams, but a stale/mismatched division fetch shouldn't crash the page).
// Goes through hideStillLiveResults first, the same reversion DivisionDetailsPage's own table
// applies (issue #82): a still-live match's own page would otherwise show its two teams'
// post-match positions - a direct spoiler - well before that match's live broadcast (shown on
// this very page) finishes pacing out to the viewer.
export function positionOf(division: DivisionDetails | undefined, teamId: number): number | undefined {
  if (!division) {
    return undefined
  }
  return hideStillLiveResults(division.classification.positions, division.previousWeekMatches).find((p) => p.teamId === teamId)?.position
}

export function fromWeekMatch(match: WeekMatch, division: DivisionDetails | undefined, t: TFunction): MatchView {
  const subtitleLabel = division
    ? `${division.name} · ${t('match.matchday', { week: match.weekNumber, total: weeksInASeason(division.classification.positions.length) })}`
    : t('division.weekChip', { week: match.weekNumber })
  return {
    homeId: match.home,
    homeName: match.homeName,
    awayId: match.away,
    awayName: match.awayName,
    homeStadiumName: match.homeStadiumName,
    rivalry: match.rivalry,
    homePosition: positionOf(division, match.home),
    awayPosition: positionOf(division, match.away),
    matchDate: match.matchDate,
    played: match.played,
    homeGoals: match.homeGoals,
    awayGoals: match.awayGoals,
    events: match.events,
    matchInstants: match.matchInstants,
    homeLineup: match.homeLineup,
    awayLineup: match.awayLineup,
    homePlayers: match.homePlayers,
    awayPlayers: match.awayPlayers,
    ratings: match.ratings,
    playerRatings: match.playerRatings,
    weather: match.weather,
    attendance: match.attendance,
    refereeName: match.refereeName,
    assistant1Name: match.assistant1Name,
    assistant2Name: match.assistant2Name,
    subtitleLabel,
    liveUrl: liveMatchPath({ kind: MATCH_KIND_LEAGUE, matchId: match.matchId }),
    queryKey: queryKeys.match(String(match.matchId)),
  }
}

// cup is undefined while the cup's details load (useCupDetails), which leaves the label out rather than
// guessing a round name; the bracket's own naming (roundName & co.) is reused so the label matches it.
export function fromCupTie(tie: CupTieDetails, cup: CupDetails | undefined, t: TFunction): MatchView {
  const penaltyWinner = cupTiePenaltyWinnerName(tie)
  let subtitleLabel: string | undefined
  if (tie.competition === 'playoff') {
    // A promotion playoff tie: one match, no cup name or round.
    subtitleLabel = t('competition.playoff')
  } else if (cup) {
    const round = roundName(t, tie.round, expectedTotalRounds(cup.ties), tie.round === 1 && isFirstRoundPlayIn(cup.ties))
    subtitleLabel = `${translateCupName(t, cup.cup.name)} · ${round}`
  }
  return {
    homeId: tie.homeTeamId ?? 0,
    homeName: localizedTeamName(t, tie.homeTeamName),
    awayId: tie.awayTeamId ?? 0,
    awayName: localizedTeamName(t, tie.awayTeamName),
    homeStadiumName: tie.homeStadiumName ?? '',
    rivalry: tie.rivalry,
    matchDate: tie.matchDate ?? '',
    played: tie.played,
    homeGoals: tie.homeGoals,
    awayGoals: tie.awayGoals,
    events: tie.events,
    matchInstants: tie.matchInstants ?? null,
    homeLineup: tie.homeLineup,
    awayLineup: tie.awayLineup,
    homePlayers: tie.homePlayers,
    awayPlayers: tie.awayPlayers,
    ratings: tie.ratings,
    playerRatings: tie.playerRatings,
    weather: tie.weather,
    attendance: tie.attendance,
    refereeName: tie.refereeName,
    assistant1Name: tie.assistant1Name,
    assistant2Name: tie.assistant2Name,
    subtitleLabel,
    liveUrl: liveMatchPath({ kind: MATCH_KIND_CUP, matchId: tie.tieId }),
    queryKey: queryKeys.cupTie(String(tie.tieId)),
    wentToPenalties: tie.wentToPenalties,
    penaltyWinnerName: penaltyWinner ? localizedTeamName(t, penaltyWinner) : undefined,
  }
}

export function fromFriendly(match: FriendlyMatch, t: TFunction): MatchView {
  const subtitleLabel =
    match.worldCupGroupName && match.worldCupRound
      ? t('match.competition.worldCupGroup', { group: match.worldCupGroupName, round: match.worldCupRound })
      : t('match.competition.friendly')
  return {
    homeId: match.homeTeamId,
    homeName: localizedTeamName(t, match.homeTeamName),
    awayId: match.awayTeamId,
    awayName: localizedTeamName(t, match.awayTeamName),
    homeStadiumName: match.homeStadiumName ?? '',
    matchDate: match.matchDate,
    played: match.played,
    homeGoals: match.homeGoals,
    awayGoals: match.awayGoals,
    events: match.events ?? null,
    matchInstants: match.matchInstants ?? null,
    homeLineup: match.homeLineup ?? null,
    awayLineup: match.awayLineup ?? null,
    homePlayers: match.homePlayers ?? null,
    awayPlayers: match.awayPlayers ?? null,
    ratings: match.ratings,
    playerRatings: match.playerRatings,
    weather: match.weather,
    attendance: match.attendance,
    refereeName: match.refereeName,
    assistant1Name: match.assistant1Name,
    assistant2Name: match.assistant2Name,
    subtitleLabel,
    liveUrl: liveMatchPath({ kind: MATCH_KIND_FRIENDLY, matchId: match.matchId }),
    queryKey: queryKeys.friendlyMatch(String(match.matchId)),
  }
}
