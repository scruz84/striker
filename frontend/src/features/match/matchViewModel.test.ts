import type { TFunction } from 'i18next'
import { describe, expect, it } from 'vitest'
import type { CupDetails, CupTie, CupTieDetails, DivisionDetails, FriendlyMatch, WeekMatch } from '../../api/types'
import i18n from '../../i18n'
import { formatDateTime } from '../../utils/formatDate'
import { matchSubtitle } from '../../utils/matchSubtitle'
import { fromCupTie, fromFriendly, fromWeekMatch, weeksInASeason } from './matchViewModel'

const t = i18n.t.bind(i18n) as unknown as TFunction
const DATE = '2026-10-21T20:00:00Z'

function weekMatch(overrides: Partial<WeekMatch> = {}): WeekMatch {
  return {
    matchId: 7,
    divisionId: 1,
    season: 1,
    weekNumber: 3,
    matchDate: DATE,
    played: true,
    inSimulation: false,
    home: 1,
    homeName: 'Home',
    homeStadiumName: 'Estadio Norte',
    away: 2,
    awayName: 'Away',
    homeGoals: 2,
    awayGoals: 1,
    events: null,
    matchInstants: null,
    homeLineup: null,
    awayLineup: null,
    homePlayers: null,
    awayPlayers: null,
    weather: 'rain',
    ...overrides,
  }
}

function division(teams: number): DivisionDetails {
  return {
    id: 1,
    season: 1,
    name: 'D',
    classification: { positions: Array.from({ length: teams }, (_, i) => ({ position: i + 1, teamId: i + 1 })) },
    previousWeekMatches: [],
    nextWeekMatches: [],
  } as unknown as DivisionDetails
}

function cupTie(overrides: Partial<CupTieDetails> = {}): CupTieDetails {
  return {
    tieId: 5,
    cupId: 9,
    round: 1,
    slotNumber: 1,
    homeTeamId: 1,
    homeTeamName: 'Home',
    awayTeamId: 2,
    awayTeamName: 'Away',
    matchDate: DATE,
    played: true,
    homeGoals: 1,
    awayGoals: 1,
    wentToPenalties: true,
    winnerTeamId: 2,
    events: null,
    homeLineup: null,
    awayLineup: null,
    homePlayers: null,
    awayPlayers: null,
    ...overrides,
  }
}

// An 8-team bracket: round 1 is the quarterfinal.
function quarterfinalCup(name = 'Copa del Rey'): CupDetails {
  const ties: CupTie[] = Array.from({ length: 4 }, (_, i) => cupTie({ tieId: i + 1, slotNumber: i + 1, homeTeamId: 2 * i + 1, awayTeamId: 2 * i + 2 }))
  return { cup: { cupId: 9, name, season: 1, completed: false }, ties } as unknown as CupDetails
}

function friendly(overrides: Partial<FriendlyMatch> = {}): FriendlyMatch {
  return {
    matchId: 3,
    homeTeamId: 1,
    homeTeamName: 'Home',
    awayTeamId: 2,
    awayTeamName: 'Away',
    matchDate: DATE,
    played: true,
    homeGoals: 0,
    awayGoals: 0,
    ...overrides,
  }
}

describe('subtitleLabel per competition', () => {
  it('league: "<division> · Matchday 3 of 38" once the division is known, the bare week before that', () => {
    expect(weeksInASeason(20)).toBe(38)
    expect(fromWeekMatch(weekMatch(), division(20), t).subtitleLabel).toBe('D · Matchday 3 of 38')
    expect(fromWeekMatch(weekMatch(), undefined, t).subtitleLabel).toBe('Week 3')
  })

  it('cup tie: the cup name and the bracket round name', () => {
    expect(fromCupTie(cupTie(), quarterfinalCup(), t).subtitleLabel).toBe('Copa del Rey · Quarterfinal')
  })

  it('cup tie: a promotion playoff tie reads "Playoff", with no cup name or round', () => {
    expect(fromCupTie({ ...cupTie(), competition: 'playoff' }, quarterfinalCup(), t).subtitleLabel).toBe(t('competition.playoff'))
  })

  it('cup tie: a known cup name is translated', () => {
    expect(fromCupTie(cupTie(), quarterfinalCup('European Cup'), t).subtitleLabel).toBe(`${t('cup.names.europe')} · Quarterfinal`)
  })

  it('cup tie: no label (the date alone) while the cup details load', () => {
    const view = fromCupTie(cupTie(), undefined, t)
    expect(view.subtitleLabel).toBeUndefined()
    expect(matchSubtitle(view.subtitleLabel, view.matchDate, view.weather)).toBe(formatDateTime(DATE))
  })

  it('friendly: "Friendly" for an ordinary one', () => {
    expect(fromFriendly(friendly(), t).subtitleLabel).toBe('Friendly')
  })

  it('friendly: the group and matchday for a World Cup group fixture', () => {
    expect(fromFriendly(friendly({ worldCupGroupName: 'B', worldCupRound: 2 }), t).subtitleLabel).toBe('World Cup · Group B · Matchday 2')
  })
})

describe('adapters carry what the viewer shows', () => {
  it('maps a league match with its weather, stadium and league positions', () => {
    const view = fromWeekMatch(weekMatch(), division(20), t)
    expect(view).toMatchObject({
      homeId: 1,
      awayId: 2,
      homeStadiumName: 'Estadio Norte',
      weather: 'rain',
      liveUrl: '/v1/teams/match/7/live',
      queryKey: ['match', '7'],
      homePosition: 1,
      awayPosition: 2,
    })
    expect(matchSubtitle(view.subtitleLabel, view.matchDate, view.weather)).toBe(`D · Matchday 3 of 38 · ${formatDateTime(DATE)} · 🌧️`)
  })

  it('maps a cup tie with its stored flavor, stadium and instants, and names the shootout winner', () => {
    const instants = [{ minute: 1 }] as CupTieDetails['matchInstants']
    const view = fromCupTie(
      cupTie({ homeStadiumName: 'Estadio Norte', weather: 'sunny', attendance: 8412, refereeName: 'Ref', matchInstants: instants }),
      quarterfinalCup(),
      t,
    )
    expect(view).toMatchObject({
      homeStadiumName: 'Estadio Norte',
      weather: 'sunny',
      attendance: 8412,
      refereeName: 'Ref',
      matchInstants: instants,
      liveUrl: '/v1/cups/ties/5/live',
      queryKey: ['cupTie', '5'],
      wentToPenalties: true,
      penaltyWinnerName: 'Away',
    })
  })

  it('a cup tie played before instants were stored has none, so the tactics tab shows only the pitch', () => {
    expect(fromCupTie(cupTie(), quarterfinalCup(), t).matchInstants).toBeNull()
  })

  it('a national team has no stadium', () => {
    expect(fromFriendly(friendly(), t).homeStadiumName).toBe('')
  })

  it('maps a friendly with its stored flavor and a played friendly before the viewer stored it', () => {
    const stored = fromFriendly(friendly({ homeStadiumName: 'Estadio Norte', weather: 'windy', attendance: 5000 }), t)
    expect(stored).toMatchObject({ homeStadiumName: 'Estadio Norte', weather: 'windy', attendance: 5000, liveUrl: '/v1/friendlies/3/live' })
    const old = fromFriendly(friendly(), t)
    expect(matchSubtitle(old.subtitleLabel, old.matchDate, old.weather)).toBe(`Friendly · ${formatDateTime(DATE)}`)
  })
})
