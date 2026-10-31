export type TeamTab = 'summary' | 'squad' | 'matches' | 'academy' | 'history'

// A national team has no academy or club history, but has its own matches tab (fixtures, results
// and form - see NationalTeamMatchesTab); a club's own page adds its academy.
export function teamTabs(isNationalTeam: boolean, isOwnTeam: boolean): TeamTab[] {
  if (isNationalTeam) return ['summary', 'matches', 'squad']
  return isOwnTeam ? ['summary', 'squad', 'academy', 'history'] : ['summary', 'squad', 'history']
}
