import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import Skeleton from '@mui/material/Skeleton'
import PublicTwoToneIcon from '@mui/icons-material/PublicTwoTone'
import { useTranslation } from 'react-i18next'
import { useCupDetails, useLatestInternationalCupId, useMe } from '../../api/hooks'
import { EmptyState } from '../../components/EmptyState'
import { PageHeader } from '../../components/PageHeader'
import { translateCupName } from '../../utils/ledgerDescription'
import { regionForCountry } from '../../components/avatar/traits'
import { CupBracket } from './CupPage'

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
    return (
      <Box>
        <PageHeader title={t('header.nav.internationalCup')} />
        <EmptyState
          icon={<PublicTwoToneIcon fontSize="inherit" />}
          title={t('cup.noInternationalCupTitle')}
          text={t('cup.noInternationalCup')}
        />
      </Box>
    )
  }

  const { cup, ties } = cupDetails

  return (
    <Box>
      <PageHeader
        title={t('cup.titleWithSeason', { name: translateCupName(t, cup.name), season: cup.season })}
        subtitle={t('cup.entryCriteria')}
      />
      {cup.completed && (
        <Chip color="success" label={t('cup.champion', { name: cup.championTeamName })} sx={{ mb: 2 }} />
      )}

      <CupBracket ties={ties} />
    </Box>
  )
}
