import type { TFunction } from 'i18next'
import {
  STATS_COMPETITION_CLUB_FRIENDLY,
  STATS_COMPETITION_DOMESTIC_CUP,
  STATS_COMPETITION_INTERNATIONAL_CUP,
  STATS_COMPETITION_LEAGUE,
  STATS_COMPETITION_NATIONAL_FRIENDLY,
  STATS_COMPETITION_PLAYOFF,
  STATS_COMPETITION_WORLD_CUP,
} from '../../api/types'
import { formatDivisionName } from '../../utils/divisionName'

const REGIONS = ['europe', 'americas', 'asia', 'africa']

// Names the competition of a stats line in the viewer's language: a league by its division name,
// a regional cup by its region, everything else by its kind.
export function competitionLabel(
  t: TFunction,
  line: { competition: number; competitionName: string; competitionRegion?: string },
): string {
  switch (line.competition) {
    case STATS_COMPETITION_LEAGUE:
      return line.competitionName ? formatDivisionName(t, line.competitionName) : t('stats.competitions.league')
    case STATS_COMPETITION_DOMESTIC_CUP:
      return t('cup.names.national')
    case STATS_COMPETITION_PLAYOFF:
      return t('stats.competitions.playoff')
    case STATS_COMPETITION_INTERNATIONAL_CUP:
      return line.competitionRegion && REGIONS.includes(line.competitionRegion)
        ? t(`cup.names.${line.competitionRegion}`)
        : t('cup.names.international')
    case STATS_COMPETITION_WORLD_CUP:
      return t('stats.competitions.worldCup')
    case STATS_COMPETITION_CLUB_FRIENDLY:
      return t('stats.competitions.clubFriendly')
    case STATS_COMPETITION_NATIONAL_FRIENDLY:
      return t('stats.competitions.nationalFriendly')
    default:
      return line.competitionName
  }
}

// An average rating with one decimal, or a dash without one.
export function formatAverageRating(rating: number | undefined): string {
  return rating === undefined ? '—' : rating.toFixed(1)
}
