import { useState } from 'react'
import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Tab from '@mui/material/Tab'
import Tabs from '@mui/material/Tabs'
import PublicOutlinedIcon from '@mui/icons-material/PublicOutlined'
import { useTranslation } from 'react-i18next'
import { useCupDetails, useLatestInternationalCupId, useMe } from '../../api/hooks'
import { EmptyState } from '../../components/EmptyState'
import { PageHeader } from '../../components/PageHeader'
import { translateCupName } from '../../utils/ledgerDescription'
import { regionForCountry, type Region } from '../../components/avatar/traits'
import { CupBracket } from './CupPage'

/** Every region with its own international club cup, in tab order (backend engine
 * internationalCupNames). */
const INTERNATIONAL_CUP_REGIONS: readonly Region[] = ['europe', 'americas', 'asia', 'africa']

/** The international-cup counterpart to CupPage (issue #64): one cup per region, entrants being
 * the tier-1 champion of each country in that region with a new champion since the previous
 * edition - see backend engine.StartRegionalCupsIfDue. A regional cup's `season` is its edition
 * number within the region. Every region's cup is visible to everyone through one tab per region;
 * the tab opened first is the logged-in user's own region (from their team's country via
 * regionForCountry, the same grouping the backend uses to decide entrants). Reuses CupPage's
 * bracket rendering outright - a tie is a tie regardless of which competition it belongs to. */
export function InternationalCupPage() {
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const ownRegion = loggedUser?.country ? regionForCountry(loggedUser.country) : undefined
  const [pickedRegion, setPickedRegion] = useState<Region | undefined>(undefined)
  const region = pickedRegion ?? ownRegion ?? 'europe'

  return (
    <Box>
      <Paper variant="outlined" sx={{ mb: 2 }}>
        <Tabs
          value={region}
          onChange={(_e, value: Region) => setPickedRegion(value)}
          variant="scrollable"
          scrollButtons="auto"
          aria-label={t('header.nav.internationalCup')}
        >
          {INTERNATIONAL_CUP_REGIONS.map((entry) => (
            <Tab key={entry} value={entry} label={t(`cup.names.${entry}`)} />
          ))}
        </Tabs>
      </Paper>
      <RegionalCup region={region} />
    </Box>
  )
}

function RegionalCup({ region }: { region: Region }) {
  const { t } = useTranslation()
  const { data: latestCup, isLoading: cupIdLoading } = useLatestInternationalCupId(region)
  const { data: cupDetails, isLoading: detailsLoading } = useCupDetails(latestCup?.cupId)

  if (cupIdLoading || detailsLoading) {
    return <Skeleton variant="rectangular" height={300} />
  }

  if (!cupDetails) {
    return (
      <Box>
        <PageHeader title={t(`cup.names.${region}`)} />
        <EmptyState
          icon={<PublicOutlinedIcon fontSize="inherit" />}
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
        title={t('cup.titleWithEdition', { name: translateCupName(t, cup.name), edition: cup.season })}
        subtitle={t('cup.entryCriteria')}
      />
      {cup.completed && (
        <Chip color="success" label={t('cup.champion', { name: cup.championTeamName })} sx={{ mb: 2 }} />
      )}

      <CupBracket ties={ties} />
    </Box>
  )
}
