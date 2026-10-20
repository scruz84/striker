import { describe, expect, it } from 'vitest'
import type { AdminUserSummary } from '../../api/types'
import { filterUsers } from './userFilter'

const user = (overrides: Partial<AdminUserSummary>): AdminUserSummary => ({
  userId: 1,
  name: 'Ada Lovelace',
  email: 'ada@example.com',
  country: 'es',
  teamId: 7,
  teamName: 'Real Betis',
  isAdmin: false,
  ...overrides,
})

const users = [
  user({}),
  user({ userId: 2, name: 'Bob Stone', email: 'bob@mail.org', country: 'en', teamId: null, teamName: '' }),
  user({ userId: 3, name: 'Cleo Adams', email: 'cleo@example.com', country: 'es', teamName: 'Sevilla FC' }),
]

describe('filterUsers', () => {
  it('keeps everyone for a blank query', () => {
    expect(filterUsers(users, '')).toHaveLength(3)
    expect(filterUsers(users, '   ')).toHaveLength(3)
  })

  it('matches name, email, country and team name case-insensitively', () => {
    expect(filterUsers(users, 'LOVELACE').map((u) => u.userId)).toEqual([1])
    expect(filterUsers(users, 'mail.org').map((u) => u.userId)).toEqual([2])
    expect(filterUsers(users, 'sevilla').map((u) => u.userId)).toEqual([3])
    expect(filterUsers(users, 'en').map((u) => u.userId)).toContain(2)
  })

  it('requires every typed word to match', () => {
    expect(filterUsers(users, 'ada es').map((u) => u.userId)).toEqual([1, 3])
    expect(filterUsers(users, 'ada betis').map((u) => u.userId)).toEqual([1])
    expect(filterUsers(users, 'ada nowhere')).toEqual([])
  })
})
