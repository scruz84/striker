import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, isUnauthorized } from './client'
import type {
  AdminLeagueInfo,
  AdminOverview,
  AdminUserSummary,
  CountryStatus,
  CupDetails,
  DevUser,
  DivisionDetails,
  FinancesResponse,
  Lineup,
  LineupPosition,
  LoanListing,
  LoggedUser,
  Group,
  GroupDetails,
  HeadToHead,
  MatchSchedule,
  NotificationPreferences,
  NotificationsResponse,
  Player,
  PlayerDetails,
  ScorerStanding,
  SeasonHistoryEntry,
  TeamDetails,
  TierConfig,
  Trainer,
  TransferListing,
  TransferOffer,
  WeekMatch,
} from './types'

export const queryKeys = {
  me: ['me'] as const,
  userTeam: ['userTeam'] as const,
  team: (id: string) => ['team', id] as const,
  teamHistory: (id: string) => ['teamHistory', id] as const,
  player: (id: string) => ['player', id] as const,
  headToHead: (teamAId: number, teamBId: number) => ['headToHead', teamAId, teamBId] as const,
  division: (id: string) => ['division', id] as const,
  topScorers: (leagueId: string) => ['topScorers', leagueId] as const,
  match: (id: string) => ['match', id] as const,
  lineup: ['lineup'] as const,
  finances: ['finances'] as const,
  transfers: ['transfers'] as const,
  transferWindow: ['transferWindow'] as const,
  offersReceived: ['offersReceived'] as const,
  offersSent: ['offersSent'] as const,
  myGroups: ['myGroups'] as const,
  group: (id: string) => ['group', id] as const,
  loans: ['loans'] as const,
  freeAgents: ['freeAgents'] as const,
  availableTrainers: ['availableTrainers'] as const,
  teamTrainer: ['teamTrainer'] as const,
  latestCup: (country: string) => ['latestCup', country] as const,
  cup: (id: string) => ['cup', id] as const,
  notifications: ['notifications'] as const,
  notificationPreferences: ['notificationPreferences'] as const,
  adminOverview: ['adminOverview'] as const,
  adminUsers: ['adminUsers'] as const,
  adminWorldConfig: ['adminWorldConfig'] as const,
  adminCountries: ['adminCountries'] as const,
  adminLeagues: ['adminLeagues'] as const,
  devLoginUsers: ['devLoginUsers'] as const,
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

export function useLogin() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (jwt: string) => api.post('/v1/login', { jwt, provider: 'google' }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.me })
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
    mutationFn: (email: string) => api.post('/v1/login', { devEmail: email, provider: 'dev' }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.me })
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
    mutationFn: ({ formation, positions }: { formation: number; positions: LineupPosition[] }) =>
      api.put('/v1/user/team/lineup', { formation, positions }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.lineup })
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

export function useUpgradeStadium() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => api.post<{ capacity: number }>('/v1/user/team/stadium/upgrade'),
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
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.transfers })
      void queryClient.invalidateQueries({ queryKey: queryKeys.userTeam })
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
    mutationFn: ({ playerId, durationSeasons }: { playerId: number; durationSeasons: number }) =>
      api.post(`/v1/user/team/players/${playerId}/loan-list`, { durationSeasons }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.loans })
      void queryClient.invalidateQueries({ queryKey: queryKeys.userTeam })
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
    queryFn: () => api.get<Player[]>('/v1/free-agents'),
    enabled,
  })
}

export function useSignFreeAgent() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (playerId: number) => api.post(`/v1/free-agents/${playerId}/sign`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.freeAgents })
      void queryClient.invalidateQueries({ queryKey: queryKeys.userTeam })
    },
  })
}

export function useRenewContract() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (playerId: number) => api.post(`/v1/user/team/players/${playerId}/renew-contract`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.userTeam })
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

export function useCupDetails(cupId: number | undefined) {
  return useQuery({
    queryKey: queryKeys.cup(String(cupId ?? '')),
    queryFn: () => api.get<CupDetails>(`/v1/cups/${cupId}`),
    enabled: cupId !== undefined,
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

export function useAdminAdvanceSeason() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => api.post('/v1/admin/advance-season'),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminOverview })
    },
  })
}

export function useAdminAdjustBudget() {
  return useMutation({
    mutationFn: ({ teamId, amount, description }: { teamId: number; amount: number; description: string }) =>
      api.post(`/v1/admin/teams/${teamId}/budget`, { amount, description }),
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

export function useMakeOffer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ playerId, amount }: { playerId: number; amount: number }) => api.post('/v1/offers', { playerId, amount }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.offersSent })
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
