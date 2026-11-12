import { describe, expect, it } from 'vitest'
import {
  EVENT_TYPE_GOAL,
  EVENT_TYPE_NO_GOAL,
  SET_PIECE_DIRECT_FREE_KICK,
  SET_PIECE_PENALTY,
  type MatchEvent,
} from '../api/types'
import { goalScorers } from './matchGoals'

function event(eventType: number, minute: number, teamOwner: number, name: string | null, setPiece?: number): MatchEvent {
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
    setPiece,
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
      home: [{ name: 'Berlana', goals: [{ minute: 3, addedMinute: 0, setPiece: 0 }, { minute: 70, addedMinute: 0, setPiece: 0 }] }],
      away: [{ name: 'Lerma', goals: [{ minute: 40, addedMinute: 0, setPiece: 0 }] }],
    })
  })

  it('keeps whether each goal was open play, a penalty or a direct free kick', () => {
    const events = [
      event(EVENT_TYPE_GOAL, 70, 1, 'Berlana', SET_PIECE_DIRECT_FREE_KICK),
      event(EVENT_TYPE_GOAL, 16, 1, 'Berlana', SET_PIECE_PENALTY),
      event(EVENT_TYPE_GOAL, 33, 1, 'Berlana'),
      event(EVENT_TYPE_GOAL, 50, 2, 'Lerma', SET_PIECE_PENALTY),
    ]
    expect(goalScorers(events, 1)).toEqual({
      home: [
        {
          name: 'Berlana',
          goals: [
            { minute: 16, addedMinute: 0, setPiece: SET_PIECE_PENALTY },
            { minute: 33, addedMinute: 0, setPiece: 0 },
            { minute: 70, addedMinute: 0, setPiece: SET_PIECE_DIRECT_FREE_KICK },
          ],
        },
      ],
      away: [{ name: 'Lerma', goals: [{ minute: 50, addedMinute: 0, setPiece: SET_PIECE_PENALTY }] }],
    })
  })

  // match-viewer "Minute labels with added time": a goal at 45+2 keeps its added minute and sorts
  // after 45 and before 46.
  it('keeps added-time goals in match order', () => {
    const events = [
      { ...event(EVENT_TYPE_GOAL, 46, 1, 'Berlana'), homeGoals: 3 },
      { ...event(EVENT_TYPE_GOAL, 45, 1, 'Berlana'), addedMinute: 2 },
      event(EVENT_TYPE_GOAL, 45, 1, 'Berlana'),
    ]
    expect(goalScorers(events, 1).home[0].goals).toEqual([
      { minute: 45, addedMinute: 0, setPiece: 0 },
      { minute: 45, addedMinute: 2, setPiece: 0 },
      { minute: 46, addedMinute: 0, setPiece: 0 },
    ])
  })
})
