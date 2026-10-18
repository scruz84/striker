import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useCupDetails, useLatestInternationalCupId, useMe } from '../../api/hooks'
import { regionForCountry } from '../../components/avatar/traits'
import { expectedTotalRounds, groupByRound, isFirstRoundPlayIn, roundName, TieRow } from './CupPage'

/** The international-cup counterpart to CupPage (issue #64): one cup per region, entrants being
 * one team per active country in that region (that country's previous season's tier-1 champion -
 * see backend engine.StartInternationalCupsForNewSeason). Reuses CupPage's bracket rendering
 * outright - a tie is a tie regardless of which competition it belongs to. The logged-in user's
 * region comes from their own team's country via regionForCountry, the same region grouping the
 * backend uses to decide entrants. */
export function InternationalCupPage() {
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const region = loggedUser?.country ? regionForCountry(loggedUser.country) : undefined
  const { data: latestCup, isLoading: cupIdLoading } = useLatestInternationalCupId(region)
  const { data: cupDetails, isLoading: detailsLoading } = useCupDetails(latestCup?.cupId)

  if (cupIdLoading || detailsLoading) {
    return <Skeleton variant="rectangular" height={300} />
  }

  if (!cupDetails) {
    return <Typography color="text.secondary">{t('cup.noInternationalCup')}</Typography>
  }

  const { cup, ties } = cupDetails
  const rounds = groupByRound(ties)
  const totalRounds = expectedTotalRounds(ties)
  const playIn = isFirstRoundPlayIn(ties)

  return (
    <Box>
      <Typography variant="h5" gutterBottom>
        {t('cup.titleWithSeason', { name: cup.name, season: cup.season })}
      </Typography>
      <Typography variant="body2" color="text.secondary" gutterBottom>
        {t('cup.entryCriteria')}
      </Typography>
      {cup.completed && (
        <Chip color="success" label={t('cup.champion', { name: cup.championTeamName })} sx={{ mb: 2 }} />
      )}

      {rounds.map(({ round, ties: roundTies }) => (
        <Paper key={round} sx={{ mb: 2, p: 2 }}>
          <Typography variant="subtitle1" gutterBottom>
            {roundName(t, round, totalRounds, round === 1 && playIn)}
          </Typography>
          <Stack spacing={1}>
            {roundTies.map((tie) => (
              <TieRow key={tie.tieId} tie={tie} />
            ))}
          </Stack>
        </Paper>
      ))}
    </Box>
  )
}
