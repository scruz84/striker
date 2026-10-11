import { describe, expect, it } from 'vitest'
import { localizedTeamName } from './nationalTeamName'

// A stub translator that just echoes the key back, so these assertions check which i18n key
// localizedTeamName resolved to rather than depending on any particular locale's wording.
const t = (key: string) => key

describe('localizedTeamName', () => {
  it('resolves a national team name to its country i18n key', () => {
    expect(localizedTeamName(t, 'Spain')).toBe('countries.es')
  })

  it('strips a legacy " National Team" suffix before resolving', () => {
    expect(localizedTeamName(t, 'Spain National Team')).toBe('countries.es')
  })

  it('leaves an ordinary club name unchanged', () => {
    expect(localizedTeamName(t, 'Real Madrid')).toBe('Real Madrid')
  })

  it('leaves a name coincidentally ending in the legacy suffix unchanged if the bare prefix is not a country', () => {
    expect(localizedTeamName(t, 'My Club National Team')).toBe('My Club National Team')
  })
})
