import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, isUnauthorized } from './client'
import { subscribeToPush, unsubscribeFromPush } from '../pushNotifications'
import type { SupportedLanguage } from '../i18n'
import type {
  AcademyOverview,
  AdminLeagueInfo,
  AdminOverview,
  AdminUserSummary,
  ContractRenewalOffer,
  CountryInfo,
  CountryLeagueDirectory,
  CountryStatus,
  CupDetails,
  CupTieDetails,
  DevUser,
  DivisionDetails,
  FinancesResponse,
  FreeAgentListing,
  FreeAgentSigningResult,
  FriendlyChallenge,
  FriendlyMatch,
  FriendlySchedule,
  LeagueTransfer,
  Lineup,
  LineupPosition,
  LineupSubstitution,
  LiveMatchesOverview,
  LoanListing,
  LoggedUser,
  Group,
  GroupDetails,
  HeadToHead,
  MatchSchedule,
  NotificationPreferences,
  NotificationsResponse,
  PlayerDetails,
  ScorerStanding,
  SearchResults,
  SeasonHistoryEntry,
  Stadium,
  TeamDetails,
  TeamMatchSummary,
  TierConfig,
  Trainer,
  TeamPlayerEvolution,
  TransferListing,
  TransferMarketStats,
  TransferOffer,
  TransferOfferResult,
  WeekMatch,
} from './types'

export const queryKeys = {
  me: ['me'] as const,
  userTeam: ['userTeam'] as const,
  team: (id: string) => ['team', id] as const,
  teamHistory: (id: string) => ['teamHistory', id] as const,
  player: (id: string) => ['player', id] as const,
  teamEvolutions: (logIds: string) => ['teamEvolutions', logIds] as const,
  headToHead: (teamAId: number, teamBId: number) => ['headToHead', teamAId, teamBId] as const,
  division: (id: string) => ['division', id] as const,
  topScorers: (leagueId: string) => ['topScorers', leagueId] as const,
  recentLeagueTransfers: (leagueId: string) => ['recentLeagueTransfers', leagueId] as const,
  leagueDirectory: ['leagueDirectory'] as const,
  match: (id: string) => ['match', id] as const,
  lineup: ['lineup'] as const,
  friendlyDefaultLineup: ['friendlyDefaultLineup'] as const,
  finances: ['finances'] as const,
  transfers: ['transfers'] as const,
  transferWindow: ['transferWindow'] as const,
  transferMarketStats: ['transferMarketStats'] as const,
  offersReceived: ['offersReceived'] as const,
  offersSent: ['offersSent'] as const,
  friendlyChallengesReceived: ['friendlyChallengesReceived'] as const,
  friendlyChallengesSent: ['friendlyChallengesSent'] as const,
  upcomingFriendlies: ['upcomingFriendlies'] as const,
  recentFriendlies: ['recentFriendlies'] as const,
  friendlySchedule: ['friendlySchedule'] as const,
  friendlyMatch: (id: string) => ['friendlyMatch', id] as const,
  friendlyLineup: (id: string) => ['friendlyLineup', id] as const,
  myGroups: ['myGroups'] as const,
  group: (id: string) => ['group', id] as const,
  loans: ['loans'] as const,
  freeAgents: ['freeAgents'] as const,
  academy: ['academy'] as const,
  availableTrainers: ['availableTrainers'] as const,
  teamTrainer: ['teamTrainer'] as const,
  latestCup: (country: string) => ['latestCup', country] as const,
  latestInternationalCup: (region: string) => ['latestInternationalCup', region] as const,
  cup: (id: string) => ['cup', id] as const,
  cupTie: (id: string) => ['cupTie', id] as const,
  cupTieLineup: (id: string) => ['cupTieLineup', id] as const,
  teamMatches: (teamId: string) => ['teamMatches', teamId] as const,
  matchLineup: (id: string) => ['matchLineup', id] as const,
  notifications: ['notifications'] as const,
  notificationPreferences: ['notificationPreferences'] as const,
  adminOverview: ['adminOverview'] as const,
  adminUsers: ['adminUsers'] as const,
  adminWorldConfig: ['adminWorldConfig'] as const,
  adminCountries: ['adminCountries'] as const,
  onboardingCountries: ['onboardingCountries'] as const,
  adminLeagues: ['adminLeagues'] as const,
  devLoginUsers: ['devLoginUsers'] as const,
  liveMatchesOverview: ['liveMatchesOverview'] as const,
  search: (query: string) => ['search', query] as const,
}

export function useMe() {
  return useQuery({
    queryKey: queryKeys.me,
    queryFn: async (): Promise<LoggedUser | null> => {
      try {
        return await api.get<LoggedUser>('/v1/auth/me')
      } catch (error) {
        if (isUnauthorized(error)) {
          return null
        }
        throw error
      }
    },
    retry: false,
    staleTime: 60_000,
  })
}

// Both login mutations resolve with the freshly-fetched LoggedUser (not just the bare JWT the
// login endpoint itself returns) so the caller can route a first-time sign-in straight to the
// onboarding country picker (loggedUser.pendingCountrySelection) instead of the dashboard -
// see LoginPage.
export function useLogin() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (jwt: string) => {
      await api.post('/v1/login', { jwt, provider: 'google' })
      const loggedUser = await api.get<LoggedUser>('/v1/auth/me')
      queryClient.setQueryData(queryKeys.me, loggedUser)
      return loggedUser
    },
  })
}

// useDevLoginUsers lists the dev-login roster - always an empty array unless the backend has
// DEV_LOGIN_ENABLED set, so callers can just check for a non-empty list rather than a separate
// "is dev login on" flag.
export function useDevLoginUsers() {
  return useQuery({
    queryKey: queryKeys.devLoginUsers,
    queryFn: () => api.get<DevUser[]>('/v1/dev-login/users'),
    staleTime: Infinity,
  })
}

export function useDevLogin() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (email: string) => {
      await api.post('/v1/login', { devEmail: email, provider: 'dev' })
      const loggedUser = await api.get<LoggedUser>('/v1/auth/me')
      queryClient.setQueryData(queryKeys.me, loggedUser)
      return loggedUser
    },
  })
}

// The countries a newly signed-in, not-yet-assigned user can start their team in - only the
// ones a world has actually been generated for (see backend's GetActiveCountries), not the
// full admin-facing KnownCountries list.
export function useOnboardingCountries() {
  return useQuery({
    queryKey: queryKeys.onboardingCountries,
    queryFn: () => api.get<CountryInfo[]>('/v1/auth/onboarding/countries'),
    staleTime: Infinity,
  })
}

export function useSelectCountry() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (country: string) => api.post<LoggedUser>('/v1/auth/onboarding/country', { country }),
    onSuccess: (loggedUser) => {
      queryClient.setQueryData(queryKeys.me, loggedUser)
    },
  })
}

export function useLogout() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => api.post('/v1/auth/logout'),
    onSuccess: () => {
      queryClient.setQueryData(queryKeys.me, null)
      void queryClient.invalidateQueries({ queryKey: queryKeys.userTeam })
    },
  })
}

export function useUserTeam(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.userTeam,
    queryFn: () => api.get<TeamDetails>('/v1/user/team'),
    enabled,
  })
}

export function useRenameTeam() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (name: string) => api.put('/v1/user/team/name', { name }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.userTeam })
    },
  })
}

export function useTeamDetails(teamId: string | undefined) {
  return useQuery({
    queryKey: queryKeys.team(teamId ?? ''),
    queryFn: () => api.get<TeamDetails>(`/v1/teams/team/${teamId}/`),
    enabled: !!teamId,
  })
}

export function useTeamHistory(teamId: string | undefined) {
  return useQuery({
    queryKey: queryKeys.teamHistory(teamId ?? ''),
    queryFn: () => api.get<SeasonHistoryEntry[]>(`/v1/teams/team/${teamId}/history`),
    enabled: !!teamId,
  })
}

export function usePlayerDetails(playerId: string | undefined) {
  return useQuery({
    queryKey: queryKeys.player(playerId ?? ''),
    queryFn: () => api.get<PlayerDetails>(`/v1/teams/player/${playerId}/`),
    enabled: !!playerId,
  })
}

// The caller's own squad's progression results - the page a NOTIFICATION_PROGRESSION notification
// links to. logIds, when given (from that notification's own link - see
// engine.updateTeamPlayersAfterMatch), scopes this to exactly the players who evolved after that
// one specific match; omitted, it falls back to every current squad player's own all-time latest
// progression run.
export function useTeamLatestEvolutions(enabled: boolean, logIds?: string) {
  return useQuery({
    queryKey: queryKeys.teamEvolutions(logIds ?? ''),
    queryFn: () =>
      api.get<TeamPlayerEvolution[]>(
        logIds ? `/v1/teams/team/evolutions?logIds=${encodeURIComponent(logIds)}` : '/v1/teams/team/evolutions'
      ),
    enabled,
  })
}

// Order-independent - the backend reports wins/goals from whichever id you pass as teamAId, so
// passing (a,b) vs (b,a) just flips which side of the result is "yours."
export function useHeadToHead(teamAId: number | undefined, teamBId: number | undefined) {
  return useQuery({
    queryKey: queryKeys.headToHead(teamAId ?? 0, teamBId ?? 0),
    queryFn: () => api.get<HeadToHead>(`/v1/teams/head-to-head/${teamAId}/${teamBId}`),
    enabled: !!teamAId && !!teamBId && teamAId !== teamBId,
  })
}

export function useDivisionDetails(divisionId: string | undefined) {
  return useQuery({
    queryKey: queryKeys.division(divisionId ?? ''),
    queryFn: () => api.get<DivisionDetails>(`/v1/teams/division/${divisionId}/`),
    enabled: !!divisionId,
  })
}

export function useTopScorers(leagueId: string | undefined) {
  return useQuery({
    queryKey: queryKeys.topScorers(leagueId ?? ''),
    queryFn: () => api.get<ScorerStanding[]>(`/v1/teams/division/${leagueId}/top-scorers`),
    enabled: !!leagueId,
  })
}

export function useRecentLeagueTransfers(leagueId: string | undefined) {
  return useQuery({
    queryKey: queryKeys.recentLeagueTransfers(leagueId ?? ''),
    queryFn: () => api.get<LeagueTransfer[]>(`/v1/teams/division/${leagueId}/recent-transfers`),
    enabled: !!leagueId,
  })
}

// Every league in the game world, grouped by country - lets a user browse into one they don't
// play in (see LeaguesPage) via the same DivisionDetails page their own division link uses.
export function useLeagueDirectory() {
  return useQuery({
    queryKey: queryKeys.leagueDirectory,
    queryFn: () => api.get<CountryLeagueDirectory[]>('/v1/teams/leagues'),
  })
}

// Powers the global search page - teams/players/coaches/stadiums by (partial) name. Disabled for
// a blank/whitespace-only query so the page doesn't fire a request before the user has typed
// anything worth searching for.
export function useSearch(query: string) {
  const trimmed = query.trim()
  return useQuery({
    queryKey: queryKeys.search(trimmed),
    queryFn: () => api.get<SearchResults>(`/v1/teams/search?q=${encodeURIComponent(trimmed)}`),
    enabled: trimmed.length > 0,
  })
}

export function useMatchDetails(matchId: string | undefined) {
  return useQuery({
    queryKey: queryKeys.match(matchId ?? ''),
    queryFn: () => api.get<WeekMatch>(`/v1/teams/match/${matchId}/`),
    enabled: !!matchId,
    // Backstops the live WebSocket (useLiveMatch): if it never delivers a "complete" message -
    // dropped connection, a missed reconnect, anything - this still notices the match finished
    // and flips MatchDetailsPage over to the played view within one poll, instead of leaving the
    // page stuck on a frozen "LIVE" scoreboard forever. Stops polling once played, and while a
    // played match's data is already in hand there's nothing left that could change.
    refetchInterval: (query) => (query.state.data?.played ? false : 5000),
  })
}

export function useLineup(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.lineup,
    queryFn: () => api.get<Lineup>('/v1/user/team/lineup'),
    enabled,
  })
}

export function useSetLineup() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      formation,
      positions,
      substitutions,
    }: {
      formation: number
      positions: LineupPosition[]
      substitutions: LineupSubstitution[]
    }) => api.put('/v1/user/team/lineup', { formation, positions, substitutions }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.lineup })
    },
  })
}

export function useFriendlyDefaultLineup(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.friendlyDefaultLineup,
    queryFn: () => api.get<Lineup>('/v1/user/team/lineup/friendly'),
    enabled,
  })
}

export function useSetFriendlyDefaultLineup() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ formation, positions }: { formation: number; positions: LineupPosition[] }) =>
      api.put('/v1/user/team/lineup/friendly', { formation, positions }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.friendlyDefaultLineup })
    },
  })
}

export function useFinances(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.finances,
    queryFn: () => api.get<FinancesResponse>('/v1/user/team/finances'),
    enabled,
  })
}

export function useSetTicketPrice() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (price: number) => api.put('/v1/user/team/stadium/ticket-price', { price }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.finances })
    },
  })
}

export function useRenameStadium() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (name: string) => api.put('/v1/user/team/stadium/name', { name }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.finances })
    },
  })
}

export function useSetSponsorDeal() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (deal: number) => api.put('/v1/user/team/sponsor-deal', { deal }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.finances })
    },
  })
}

export function useUpgradeStadium() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (seats: number) => api.post<Stadium>('/v1/user/team/stadium/upgrade', { seats }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.finances })
    },
  })
}

export function useTransferListings(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.transfers,
    queryFn: () => api.get<TransferListing[]>('/v1/transfers'),
    enabled,
  })
}

export function useListPlayer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ playerId, askingPrice }: { playerId: number; askingPrice: number }) =>
      api.post(`/v1/user/team/players/${playerId}/list`, { askingPrice }),
    onSuccess: (_data, { playerId }) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.transfers })
      void queryClient.invalidateQueries({ queryKey: queryKeys.userTeam })
      void queryClient.invalidateQueries({ queryKey: queryKeys.player(String(playerId)) })
    },
  })
}

export function useUnlistPlayer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (playerId: number) => api.delete(`/v1/user/team/players/${playerId}/list`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.transfers })
    },
  })
}

export function useBuyListing() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (listingId: number) => api.post(`/v1/transfers/${listingId}/buy`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.transfers })
      void queryClient.invalidateQueries({ queryKey: queryKeys.userTeam })
      void queryClient.invalidateQueries({ queryKey: queryKeys.finances })
    },
  })
}

export function useLoanListings(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.loans,
    queryFn: () => api.get<LoanListing[]>('/v1/loans'),
    enabled,
  })
}

export function useListPlayerForLoan() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      playerId,
      durationSeasons,
      loanFee,
      wageContributionPercent,
      buyOptionFee,
    }: {
      playerId: number
      durationSeasons: number
      loanFee?: number
      wageContributionPercent?: number
      buyOptionFee?: number
    }) =>
      api.post(`/v1/user/team/players/${playerId}/loan-list`, {
        durationSeasons,
        loanFee: loanFee ?? 0,
        wageContributionPercent: wageContributionPercent ?? 0,
        buyOptionFee: buyOptionFee ?? 0,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.loans })
      void queryClient.invalidateQueries({ queryKey: queryKeys.userTeam })
    },
  })
}

// Lets the caller's team make permanent a loaned-in player's move by paying the loan's
// buy-option fee (see player.onLoanBuyOptionFee) - same money movement as buying a listed
// player, so it refreshes the same queries useBuyListing does.
export function useExerciseLoanBuyOption() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (playerId: number) => api.post(`/v1/user/team/players/${playerId}/exercise-loan-buy-option`),
    onSuccess: (_data, playerId) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.userTeam })
      void queryClient.invalidateQueries({ queryKey: queryKeys.finances })
      void queryClient.invalidateQueries({ queryKey: queryKeys.player(String(playerId)) })
    },
  })
}

export function useUnlistPlayerForLoan() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (playerId: number) => api.delete(`/v1/user/team/players/${playerId}/loan-list`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.loans })
    },
  })
}

export function useAcceptLoan() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (listingId: number) => api.post(`/v1/loans/${listingId}/accept`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.loans })
      void queryClient.invalidateQueries({ queryKey: queryKeys.userTeam })
    },
  })
}

export function useFreeAgents(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.freeAgents,
    queryFn: () => api.get<FreeAgentListing[]>('/v1/free-agents'),
    enabled,
  })
}

// The player evaluates the offered wage/years immediately (see backend's SignFreeAgent) and the
// response always carries a verdict, accepted or not - a decline isn't a mutation error, it's a
// normal 200 with accepted: false, so the caller can show the player's actual expectations and
// let the team retry with different terms. A transfer premium (if any) is included when accepted.
export function useSignFreeAgent() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ playerId, wage, years }: { playerId: number; wage: number; years: number }) =>
      api.post<FreeAgentSigningResult>(`/v1/free-agents/${playerId}/sign`, { wage, years }),
    onSuccess: (result) => {
      if (result.accepted) {
        void queryClient.invalidateQueries({ queryKey: queryKeys.freeAgents })
        void queryClient.invalidateQueries({ queryKey: queryKeys.userTeam })
      }
    },
  })
}

// The player evaluates the offer immediately (see backend's MakeContractRenewalOffer) and the
// response always carries a verdict, accepted or not - a decline isn't a mutation error, it's a
// normal 200 with accepted: false, so the caller can show the player's actual expectations and
// let the team retry with different terms.
export function useRenewContract() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ playerId, wage, years }: { playerId: number; wage: number; years: number }) =>
      api.post<ContractRenewalOffer>(`/v1/user/team/players/${playerId}/renew-contract`, { wage, years }),
    onSuccess: (offer) => {
      if (offer.accepted) {
        void queryClient.invalidateQueries({ queryKey: queryKeys.userTeam })
      }
    },
  })
}

export function useReleasePlayer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (playerId: number) => api.post(`/v1/user/team/players/${playerId}/release`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.userTeam })
      void queryClient.invalidateQueries({ queryKey: queryKeys.finances })
      void queryClient.invalidateQueries({ queryKey: queryKeys.freeAgents })
    },
  })
}

// The caller's own team's youth academy ("cantera") pool - a manager may promote at most one
// prospect per real calendar week (see usePromoteFromAcademy and backend common.PromoteFromAcademy).
export function useAcademy() {
  return useQuery({
    queryKey: queryKeys.academy,
    queryFn: () => api.get<AcademyOverview>('/v1/user/team/academy'),
  })
}

export function usePromoteFromAcademy() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (playerId: number) => api.post(`/v1/user/team/academy/${playerId}/promote`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.academy })
      void queryClient.invalidateQueries({ queryKey: queryKeys.userTeam })
      void queryClient.invalidateQueries({ queryKey: queryKeys.finances })
    },
  })
}

export function useAvailableTrainers(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.availableTrainers,
    queryFn: () => api.get<Trainer[]>('/v1/trainers'),
    enabled,
  })
}

export function useTeamTrainer(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.teamTrainer,
    queryFn: () => api.get<Trainer | null>('/v1/trainers/mine'),
    enabled,
  })
}

export function useHireTrainer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (trainerId: number) => api.post(`/v1/trainers/${trainerId}/hire`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.availableTrainers })
      void queryClient.invalidateQueries({ queryKey: queryKeys.teamTrainer })
    },
  })
}

export function useFireTrainer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (trainerId: number) => api.post(`/v1/trainers/${trainerId}/fire`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.availableTrainers })
      void queryClient.invalidateQueries({ queryKey: queryKeys.teamTrainer })
    },
  })
}

export function useLatestCupId(country: string | undefined) {
  return useQuery({
    queryKey: queryKeys.latestCup(country ?? ''),
    queryFn: () => api.get<{ cupId: number }>(`/v1/cups/latest/${country}`),
    enabled: !!country,
  })
}

// LatestCupId's international-cup counterpart (issue #64) - region comes from regionForCountry
// (see components/avatar/traits.ts), applied to the logged-in user's own country.
export function useLatestInternationalCupId(region: string | undefined) {
  return useQuery({
    queryKey: queryKeys.latestInternationalCup(region ?? ''),
    queryFn: () => api.get<{ cupId: number }>(`/v1/cups/latest-international/${region}`),
    enabled: !!region,
  })
}

export function useCupDetails(cupId: number | undefined) {
  return useQuery({
    queryKey: queryKeys.cup(String(cupId ?? '')),
    queryFn: () => api.get<CupDetails>(`/v1/cups/${cupId}`),
    enabled: cupId !== undefined,
  })
}

export function useCupTieDetails(tieId: string | undefined) {
  return useQuery({
    queryKey: queryKeys.cupTie(tieId ?? ''),
    queryFn: () => api.get<CupTieDetails>(`/v1/cups/ties/${tieId}`),
    enabled: !!tieId,
    // Same backstop reasoning as useMatchDetails - stop polling once played.
    refetchInterval: (query) => (query.state.data?.played ? false : 5000),
  })
}

export function useCupTieLineup(tieId: string | undefined, enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.cupTieLineup(tieId ?? ''),
    queryFn: () => api.get<Lineup>(`/v1/cups/ties/${tieId}/lineup`),
    enabled: enabled && !!tieId,
  })
}

export function useSetCupTieLineup(tieId: string | undefined) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      formation,
      positions,
      substitutions,
    }: {
      formation: number
      positions: LineupPosition[]
      substitutions: LineupSubstitution[]
    }) => api.put(`/v1/cups/ties/${tieId}/lineup`, { formation, positions, substitutions }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.cupTieLineup(tieId ?? '') })
    },
  })
}

// useTeamMatches lists a team's full fixture/results history across league, cup, and friendly
// matches, marked by kind - see backend/common.GetTeamMatches.
export function useTeamMatches(teamId: string | undefined) {
  return useQuery({
    queryKey: queryKeys.teamMatches(teamId ?? ''),
    queryFn: () => api.get<TeamMatchSummary[]>(`/v1/teams/team/${teamId}/matches`),
    enabled: !!teamId,
  })
}

// useLiveMatchesOverview polls how many matches are live across the whole world right now, plus
// a suggested one to watch (prioritized toward the top divisions) - see backend
// common.GetLiveMatchesOverview. The 5s refetch interval doubles as the dashboard's auto-rotation
// through suggested matches (each poll asks the backend for a fresh random pick, same as the
// shuffle button) and keeps the suggested match's minute/score-as-of-minute reasonably current.
export function useLiveMatchesOverview(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.liveMatchesOverview,
    queryFn: () => api.get<LiveMatchesOverview>('/v1/teams/matches/live'),
    enabled,
    refetchInterval: 5000,
  })
}

export function useMatchLineup(matchId: string | undefined, enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.matchLineup(matchId ?? ''),
    queryFn: () => api.get<Lineup>(`/v1/teams/match/${matchId}/lineup`),
    enabled: enabled && !!matchId,
  })
}

export function useSetMatchLineup(matchId: string | undefined) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      formation,
      positions,
      substitutions,
    }: {
      formation: number
      positions: LineupPosition[]
      substitutions: LineupSubstitution[]
    }) => api.put(`/v1/teams/match/${matchId}/lineup`, { formation, positions, substitutions }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.matchLineup(matchId ?? '') })
    },
  })
}

// Polled rather than pushed - there's no live match/event stream yet, so this is the simplest
// way to keep the unread badge reasonably fresh while a user is on the page.
const NOTIFICATIONS_POLL_INTERVAL_MS = 30_000

export function useNotifications(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.notifications,
    queryFn: () => api.get<NotificationsResponse>('/v1/user/notifications'),
    enabled,
    refetchInterval: enabled ? NOTIFICATIONS_POLL_INTERVAL_MS : false,
  })
}

export function useMarkNotificationRead() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (notificationId: number) => api.post(`/v1/user/notifications/${notificationId}/read`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.notifications })
    },
  })
}

export function useMarkAllNotificationsRead() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => api.post('/v1/user/notifications/read-all'),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.notifications })
    },
  })
}

export function useDeleteNotification() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (notificationId: number) => api.delete(`/v1/user/notifications/${notificationId}`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.notifications })
    },
  })
}

export function useDeleteAllNotifications() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => api.delete('/v1/user/notifications'),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.notifications })
    },
  })
}

export function useNotificationPreferences(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.notificationPreferences,
    queryFn: () => api.get<NotificationPreferences>('/v1/user/notification-preferences'),
    enabled,
  })
}

export function useSetNotificationPreferences() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (prefs: NotificationPreferences) => api.put('/v1/user/notification-preferences', prefs),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.notificationPreferences })
    },
  })
}

// Syncs a logged-in user's chosen UI language to the backend (see LanguageSwitcher), so
// email/push notification delivery can render in that language too - see
// common.deliverExternalNotifications. Fire-and-forget: nothing client-side depends on the
// result, so failures aren't surfaced beyond the console.
export function useSetUserLanguage() {
  return useMutation({
    mutationFn: (language: SupportedLanguage) => api.put('/v1/user/language', { language }),
  })
}

// Requests notification permission and registers this browser for push (see
// pushNotifications.subscribeToPush) - the caller (NotificationBell) only flips the pushEnabled
// preference on once this resolves, so a denied permission or a subscribe failure leaves it off.
export function useSubscribeToPush() {
  return useMutation({
    mutationFn: async () => {
      const subscription = await subscribeToPush()
      return api.post('/v1/user/push-subscriptions', subscription)
    },
  })
}

// Unsubscribes this browser from push and tells the backend to drop it (see
// pushNotifications.unsubscribeFromPush) - a no-op, but not an error, if it was never subscribed
// in the first place.
export function useUnsubscribeFromPush() {
  return useMutation({
    mutationFn: async () => {
      const endpoint = await unsubscribeFromPush()
      if (!endpoint) {
        return
      }
      await api.delete('/v1/user/push-subscriptions', { endpoint })
    },
  })
}

export function useAdminOverview(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.adminOverview,
    queryFn: () => api.get<AdminOverview>('/v1/admin/overview'),
    enabled,
  })
}

export function useAdminUsers(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.adminUsers,
    queryFn: () => api.get<AdminUserSummary[]>('/v1/admin/users'),
    enabled,
  })
}

export function useSetUserAdmin() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, isAdmin }: { userId: number; isAdmin: boolean }) =>
      api.put(`/v1/admin/users/${userId}/admin`, { isAdmin }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminUsers })
    },
  })
}

export function useAdminWorldConfig(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.adminWorldConfig,
    queryFn: () => api.get<{ tiers: TierConfig[] }>('/v1/admin/world-config'),
    enabled,
  })
}

export function useSetWorldConfig() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (tiers: TierConfig[]) => api.put('/v1/admin/world-config', { tiers }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminWorldConfig })
    },
  })
}

export function useAdminRecreateLeagues() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => api.post('/v1/admin/recreate-leagues'),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminOverview })
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminUsers })
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminCountries })
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminLeagues })
    },
  })
}

export function useAdminCountries(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.adminCountries,
    queryFn: () => api.get<CountryStatus[]>('/v1/admin/countries'),
    enabled,
  })
}

export function useAdminAddCountry() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (code: string) => api.post('/v1/admin/countries', { code }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminCountries })
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminOverview })
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminLeagues })
    },
  })
}

export function useAdminLeagues(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.adminLeagues,
    queryFn: () => api.get<AdminLeagueInfo[]>('/v1/admin/leagues'),
    enabled,
  })
}

// Adds one world-config tier's division to a country that's already active (issue #68) - either
// with this season's fixtures generated and backfilled to match the rest of the country
// (withMatches: true), or as teams only, joining at the next season rollover like every other
// league does (withMatches: false). See handlers.AdminAddDivisionHandler.
export function useAdminAddDivision() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ country, tier, withMatches }: { country: string; tier: number; withMatches: boolean }) =>
      api.post(`/v1/admin/countries/${country}/divisions`, { tier, withMatches }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminLeagues })
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminOverview })
    },
  })
}

export function useSetLeagueSchedule() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ leagueId, schedule }: { leagueId: number; schedule: MatchSchedule }) =>
      api.put(`/v1/admin/leagues/${leagueId}/schedule`, schedule),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminLeagues })
    },
  })
}

export function useClearLeagueSchedule() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (leagueId: number) => api.delete(`/v1/admin/leagues/${leagueId}/schedule`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminLeagues })
    },
  })
}

export function useAdminSimulateNow() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => api.post('/v1/admin/simulate'),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminOverview })
    },
  })
}

export function useAdminFinishStuckMatches() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () =>
      api.post<{ leagueMatches: number; friendlies: number }>('/v1/admin/finish-stuck-matches'),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminOverview })
    },
  })
}

export function useAdminAdvanceSeason() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => api.post('/v1/admin/advance-season'),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminOverview })
    },
  })
}

export function useAdminPayWagesNow() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => api.post('/v1/admin/pay-wages'),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminOverview })
    },
  })
}

export function useAdminInjectFreeAgentsNow() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => api.post('/v1/admin/inject-free-agents'),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminOverview })
      void queryClient.invalidateQueries({ queryKey: queryKeys.freeAgents })
    },
  })
}

export function useAdminAdjustBudget() {
  return useMutation({
    mutationFn: ({ teamId, amount, description }: { teamId: number; amount: number; description: string }) =>
      api.post(`/v1/admin/teams/${teamId}/budget`, { amount, description }),
  })
}

export function useAdminInjectAllTeamsBudget() {
  return useMutation({
    mutationFn: ({ amount, description }: { amount: number; description: string }) =>
      api.post<{ message: string; teamsAffected: number }>('/v1/admin/teams/inject-all', { amount, description }),
  })
}

export function useAdminSetAiLevel() {
  return useMutation({
    mutationFn: ({ teamId, aiLevel }: { teamId: number; aiLevel: number }) =>
      api.put(`/v1/admin/teams/${teamId}/ai-level`, { aiLevel }),
  })
}

export function useTransferWindowStatus(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.transferWindow,
    queryFn: () => api.get<{ open: boolean }>('/v1/transfers/window'),
    enabled,
  })
}

// The transfer market's activity panel (issue #72) - see backend common.GetTransferMarketStats.
export function useTransferMarketStats(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.transferMarketStats,
    queryFn: () => api.get<TransferMarketStats>('/v1/transfers/stats'),
    enabled,
    // The offers-in-the-last-hour/24h counters age out purely with the passage of time, not with
    // any action the viewer takes on this page - without a poll, a manager who parks on the
    // transfer market page would see a stale "1 last hour" long after that offer has aged out.
    refetchInterval: 60_000,
  })
}

export function useOffersReceived(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.offersReceived,
    queryFn: () => api.get<TransferOffer[]>('/v1/offers/received'),
    enabled,
  })
}

export function useOffersSent(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.offersSent,
    queryFn: () => api.get<TransferOffer[]>('/v1/offers/sent'),
    enabled,
  })
}

// The player evaluates the offered personal terms (wage/years) immediately, before the offer
// ever reaches the selling club (see backend's MakeTransferOffer) - a decline isn't a mutation
// error, it's a normal 200 with accepted: false, so the caller can show the player's actual
// expectations and let the buyer retry with better terms.
export function useMakeOffer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ playerId, amount, wage, years }: { playerId: number; amount: number; wage: number; years: number }) =>
      api.post<TransferOfferResult>('/v1/offers', { playerId, amount, wage, years }),
    onSuccess: (result) => {
      if (result.accepted) {
        void queryClient.invalidateQueries({ queryKey: queryKeys.offersSent })
      }
    },
  })
}

export function useWithdrawOffer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (offerId: number) => api.delete(`/v1/offers/${offerId}`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.offersSent })
    },
  })
}

export function useRejectOffer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (offerId: number) => api.post(`/v1/offers/${offerId}/reject`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.offersReceived })
    },
  })
}

export function useAcceptOffer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (offerId: number) => api.post(`/v1/offers/${offerId}/accept`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.offersReceived })
      void queryClient.invalidateQueries({ queryKey: queryKeys.userTeam })
      void queryClient.invalidateQueries({ queryKey: queryKeys.finances })
      // Accepting an offer also drops the player's transfer/loan listing server-side.
      void queryClient.invalidateQueries({ queryKey: queryKeys.transfers })
      void queryClient.invalidateQueries({ queryKey: queryKeys.loans })
    },
  })
}

export function useFriendlyChallengesReceived(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.friendlyChallengesReceived,
    queryFn: () => api.get<FriendlyChallenge[]>('/v1/friendlies/challenges/received'),
    enabled,
  })
}

export function useFriendlyChallengesSent(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.friendlyChallengesSent,
    queryFn: () => api.get<FriendlyChallenge[]>('/v1/friendlies/challenges/sent'),
    enabled,
  })
}

export function useUpcomingFriendlies(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.upcomingFriendlies,
    queryFn: () => api.get<FriendlyMatch[]>('/v1/friendlies/upcoming'),
    enabled,
  })
}

export function useRecentFriendlies(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.recentFriendlies,
    queryFn: () => api.get<FriendlyMatch[]>('/v1/friendlies/recent'),
    enabled,
  })
}

export function useChallengeToFriendly() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      opponentTeamId,
      proposedKickoff,
      challengerIsHome,
    }: {
      opponentTeamId: number
      proposedKickoff?: string
      challengerIsHome?: boolean
    }) => api.post('/v1/friendlies/challenges', { opponentTeamId, proposedKickoff, challengerIsHome }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.friendlyChallengesSent })
    },
  })
}

export function useAcceptFriendlyChallenge() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ challengeId, kickoff }: { challengeId: number; kickoff: string }) =>
      api.post(`/v1/friendlies/challenges/${challengeId}/accept`, { kickoff }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.friendlyChallengesReceived })
      void queryClient.invalidateQueries({ queryKey: queryKeys.upcomingFriendlies })
    },
  })
}

export function useFriendlySchedule(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.friendlySchedule,
    queryFn: () => api.get<FriendlySchedule>('/v1/friendlies/schedule'),
    enabled,
  })
}

export function useFriendlyMatch(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.friendlyMatch(id ?? ''),
    queryFn: () => api.get<FriendlyMatch>(`/v1/friendlies/${id}`),
    enabled: !!id,
  })
}

export function useFriendlyLineup(id: string | undefined, enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.friendlyLineup(id ?? ''),
    queryFn: () => api.get<Lineup>(`/v1/friendlies/${id}/lineup`),
    enabled: enabled && !!id,
  })
}

export function useSetFriendlyLineup(id: string | undefined) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      formation,
      positions,
      substitutions,
    }: {
      formation: number
      positions: LineupPosition[]
      substitutions: LineupSubstitution[]
    }) => api.put(`/v1/friendlies/${id}/lineup`, { formation, positions, substitutions }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.friendlyLineup(id ?? '') })
    },
  })
}

export function useRejectFriendlyChallenge() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (challengeId: number) => api.post(`/v1/friendlies/challenges/${challengeId}/reject`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.friendlyChallengesReceived })
    },
  })
}

export function useWithdrawFriendlyChallenge() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (challengeId: number) => api.delete(`/v1/friendlies/challenges/${challengeId}`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.friendlyChallengesSent })
    },
  })
}

export function useAdminSetMaxFriendliesPerWeek() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (maxFriendliesPerWeek: number) => api.put('/v1/admin/max-friendlies-per-week', { maxFriendliesPerWeek }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminOverview })
    },
  })
}

export function useAdminSetFreeAgentInjectionsPerWeek() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (freeAgentInjectionsPerWeek: number) => api.put('/v1/admin/free-agent-injections-per-week', { freeAgentInjectionsPerWeek }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminOverview })
    },
  })
}

export function useAdminSetOfferNegotiationWindow() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (offerNegotiationWindowMinutes: number) => api.put('/v1/admin/offer-negotiation-window', { offerNegotiationWindowMinutes }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminOverview })
    },
  })
}

export function useAdminSetFriendlySchedule() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (weekdaysMask: number) => api.put('/v1/admin/friendly-schedule', { weekdaysMask }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminOverview })
    },
  })
}

export function useAdminSetPromotionRelegation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (config: { promotionSlotsPerGroup: number; relegationSlotsPerGroup: number }) =>
      api.put('/v1/admin/promotion-relegation', config),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminOverview })
    },
  })
}

export function useSetTransferWindow() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (open: boolean) => api.put('/v1/admin/transfer-window', { open }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.transferWindow })
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminOverview })
    },
  })
}

export function useSetMatchSchedule() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (schedule: { weekdaysMask: number; hour: number; minute: number }) =>
      api.put('/v1/admin/match-schedule', schedule),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminOverview })
    },
  })
}

export function useMyGroups(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.myGroups,
    queryFn: () => api.get<Group[]>('/v1/groups/mine'),
    enabled,
  })
}

export function useGroupDetails(groupId: string | undefined, enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.group(groupId ?? ''),
    queryFn: () => api.get<GroupDetails>(`/v1/groups/${groupId}`),
    enabled: enabled && !!groupId,
  })
}

export function useCreateGroup() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (name: string) => api.post<Group>('/v1/groups', { name }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.myGroups })
    },
  })
}

export function useJoinGroup() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (inviteCode: string) => api.post<Group>('/v1/groups/join', { inviteCode }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.myGroups })
    },
  })
}

export function useLeaveGroup() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (groupId: number) => api.post(`/v1/groups/${groupId}/leave`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.myGroups })
    },
  })
}

export function useDeleteGroup() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (groupId: number) => api.delete(`/v1/groups/${groupId}`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.myGroups })
    },
  })
}
