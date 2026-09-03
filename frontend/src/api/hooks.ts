import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, isUnauthorized } from './client'
import type {
  DivisionDetails,
  FinancesResponse,
  Lineup,
  LineupPosition,
  LoggedUser,
  TeamDetails,
  TransferListing,
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
