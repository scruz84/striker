import { MATCH_KIND_CUP, MATCH_KIND_LEAGUE, type TeamMatchSummary } from '../api/types'

// How a TeamMatchSummary (see useTeamMatches) links, colors, and labels by competition kind -
// shared by the "all my matches" list and the dashboard's next/last match cards so a league, cup,
// or friendly match always routes and reads the same way everywhere it's shown.
export function matchKindColor(kind: TeamMatchSummary['kind']): 'primary' | 'secondary' | 'default' {
  switch (kind) {
    case MATCH_KIND_LEAGUE:
      return 'primary'
    case MATCH_KIND_CUP:
      return 'secondary'
    default:
      return 'default'
  }
}

export function matchKindLabelKey(kind: TeamMatchSummary['kind']): string {
  switch (kind) {
    case MATCH_KIND_LEAGUE:
      return 'team.matches.kindLeague'
    case MATCH_KIND_CUP:
      return 'team.matches.kindCup'
    default:
      return 'team.matches.kindFriendly'
  }
}

export function matchSummaryLink(match: Pick<TeamMatchSummary, 'kind' | 'matchId'>): string {
  switch (match.kind) {
    case MATCH_KIND_LEAGUE:
      return `/match/${match.matchId}`
    case MATCH_KIND_CUP:
      return `/cup/ties/${match.matchId}`
    default:
      return `/friendlies/${match.matchId}`
  }
}

// WebSocket path of a match's live broadcast, by competition kind - the match viewer and the
// dashboard's live hero both subscribe to it, so they always agree on score and minute.
export function liveMatchPath(match: Pick<TeamMatchSummary, 'kind' | 'matchId'>): string {
  switch (match.kind) {
    case MATCH_KIND_LEAGUE:
      return `/v1/teams/match/${match.matchId}/live`
    case MATCH_KIND_CUP:
      return `/v1/cups/ties/${match.matchId}/live`
    default:
      return `/v1/friendlies/${match.matchId}/live`
  }
}
