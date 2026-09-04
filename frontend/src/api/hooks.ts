import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, isUnauthorized } from './client'
import type {
  AdminOverview,
  AdminUserSummary,
  CupDetails,
  DivisionDetails,
  FinancesResponse,
  Lineup,
  LineupPosition,
  LoanListing,
  LoggedUser,
  NotificationsResponse,
  TeamDetails,
  TransferListing,
  TransferOffer,
  WeekMatch,
} from './types'

export const queryKeys = {
  me: ['me'] as const,
  userTeam: ['userTeam'] as const,
  team: (id: string) => ['team', id] as const,
  division: (id: string) => ['division', id] as const,
  match: (id: string) => ['match', id] as const,
  lineup: ['lineup'] as const,
  finances: ['finances'] as const,
  transfers: ['transfers'] as const,
  transferWindow: ['transferWindow'] as const,
  offersReceived: ['offersReceived'] as const,
  offersSent: ['offersSent'] as const,
  loans: ['loans'] as const,
  latestCup: (country: string) => ['latestCup', country] as const,
  cup: (id: string) => ['cup', id] as const,
  notifications: ['notifications'] as const,
  adminOverview: ['adminOverview'] as const,
  adminUsers: ['adminUsers'] as const,
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

export function useTeamDetails(teamId: string | undefined) {
  return useQuery({
    queryKey: queryKeys.team(teamId ?? ''),
    queryFn: () => api.get<TeamDetails>(`/v1/teams/team/${teamId}/`),
    enabled: !!teamId,
  })
}

export function useDivisionDetails(divisionId: string | undefined) {
  return useQuery({
    queryKey: queryKeys.division(divisionId ?? ''),
    queryFn: () => api.get<DivisionDetails>(`/v1/teams/division/${divisionId}/`),
    enabled: !!divisionId,
  })
}

export function useMatchDetails(matchId: string | undefined) {
  return useQuery({
    queryKey: queryKeys.match(matchId ?? ''),
    queryFn: () => api.get<WeekMatch>(`/v1/teams/match/${matchId}/`),
    enabled: !!matchId,
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
    mutationFn: (positions: LineupPosition[]) => api.put('/v1/user/team/lineup', { positions }),
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
