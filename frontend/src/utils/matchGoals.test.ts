import { describe, expect, it } from 'vitest'
import { EVENT_TYPE_GOAL, EVENT_TYPE_NO_GOAL, type MatchEvent } from '../api/types'
import { goalScorers } from './matchGoals'

function event(eventType: number, minute: number, teamOwner: number, name: string | null): MatchEvent {
  return {
    eventType,
    minute,
    teamOwner,
    teamOwnerPlayers: name ? [1] : null,
    teamOwnerPlayerNames: name ? [name] : null,
    otherTeam: 0,
    otherTeamPlayers: null,
    otherTeamPlayerNames: null,
    homeGoals: 0,
    awayGoals: 0,
  }
}

describe('goalScorers', () => {
  it('groups goals by side and scorer, in minute order, ignoring misses and nameless goals', () => {
    const events = [
      event(EVENT_TYPE_GOAL, 40, 2, 'Lerma'),
      event(EVENT_TYPE_GOAL, 3, 1, 'Berlana'),
      event(EVENT_TYPE_NO_GOAL, 10, 1, 'Berlana'),
      event(EVENT_TYPE_GOAL, 70, 1, 'Berlana'),
      event(EVENT_TYPE_GOAL, 80, 2, null),
    ]
    expect(goalScorers(events, 1)).toEqual({
      home: [{ name: 'Berlana', minutes: [3, 70] }],
      away: [{ name: 'Lerma', minutes: [40] }],
    })
  })
})
