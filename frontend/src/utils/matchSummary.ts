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
