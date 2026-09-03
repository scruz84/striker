import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, isUnauthorized } from './client'
import type { DivisionDetails, Lineup, LineupPosition, LoggedUser, TeamDetails, WeekMatch } from './types'

export const queryKeys = {
  me: ['me'] as const,
  userTeam: ['userTeam'] as const,
  team: (id: string) => ['team', id] as const,
  division: (id: string) => ['division', id] as const,
  match: (id: string) => ['match', id] as const,
  lineup: ['lineup'] as const,
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
