import type { AdminUserSummary } from '../../api/types'

// filterUsers keeps the users whose name, email, country or team name contains every word typed
// (case-insensitive), so "ada es" finds a user named Ada in Spain. A blank query keeps everyone.
export function filterUsers(users: AdminUserSummary[], query: string): AdminUserSummary[] {
  const words = query.toLowerCase().split(/\s+/).filter(Boolean)
  if (words.length === 0) {
    return users
  }
  return users.filter((user) => {
    const haystack = [user.name, user.email, user.country, user.teamName].join(' ').toLowerCase()
    return words.every((word) => haystack.includes(word))
  })
}
