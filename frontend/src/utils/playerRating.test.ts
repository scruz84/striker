import { describe, expect, it } from 'vitest'
import { POSITION_GOALKEEPER, POSITION_STRIKER, type Player } from '../api/types'
import { bestCorePosition, ratingTier } from './playerRating'

function player(overrides: Partial<Player>): Player {
  return {
    id: 1,
    teamId: 1,
    firstName: 'P',
    lastName: '1',
    birthDay: '2000-01-01',
    goalKeeper: 0,
    defense: 0,
    lateral: 0,
    pass: 0,
    midfield: 0,
    dribbling: 0,
    shot: 0,
    header: false,
    quick: false,
    creative: false,
    technical: false,
    wage: 0,
    age: 25,
    contractYearsRemaining: 1,
    stamina: 50,
    form: 50,
    ...overrides,
  }
}

describe('bestCorePosition', () => {
  it('picks Striker for a finisher with weak Pass, mirroring the backend\'s shot-weighted formula', () => {
    // The motivating case: Shot/Dribbling clearly say "striker", but a plain three-way average of
    // Shot+Pass+Dribbling used to drag the value below this player's Goalkeeper stat (55) - see
    // backend common.GetStrikerValue, which this mirrors by hand.
    const finisher = player({ goalKeeper: 55, defense: 50, shot: 64, dribbling: 60, pass: 34, midfield: 58 })

    expect(bestCorePosition(finisher)).toBe(POSITION_STRIKER)
  })

  it('still picks Goalkeeper when no outfield position value beats it', () => {
    const keeper = player({ goalKeeper: 70, defense: 40, shot: 30, dribbling: 30, pass: 30, midfield: 30 })

    expect(bestCorePosition(keeper)).toBe(POSITION_GOALKEEPER)
  })
})

describe('ratingTier', () => {
  it('maps a 0-99 rating onto five ascending tiers', () => {
    expect([0, 34, 35, 49, 50, 64, 65, 79, 80, 99].map(ratingTier)).toEqual([0, 0, 1, 1, 2, 2, 3, 3, 4, 4])
  })
})
