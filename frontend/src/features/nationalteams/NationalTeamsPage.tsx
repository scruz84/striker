import Box from '@mui/material/Box'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Tab from '@mui/material/Tab'
import Tabs from '@mui/material/Tabs'
import { useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { PageHeader } from '../../components/PageHeader'
import { ResultsTab } from './ResultsTab'
import { RoundTab } from './RoundTab'
import { WorldCupTab } from './WorldCupTab'
import { nationalTeamsTabFromParam, type NationalTeamsTab } from './nationalMatches'

/** Where a manager follows the national teams: the current international round (every live,
 * upcoming and just-finished match - see backend common.GetNationalTeamRound), the full results
 * history, and the World Cup. Every score shown is broadcast-safe: the backend never sends the
 * final result of a match whose live replay is still running. */
export function NationalTeamsPage() {
  const { t } = useTranslation()
  const [searchParams, setSearchParams] = useSearchParams()
  const tab = nationalTeamsTabFromParam(searchParams.get('tab'))

  return (
    <Stack spacing={2}>
      <PageHeader title={t('nationalTeams.title')} subtitle={t('nationalTeams.subtitle')} />

      <Paper variant="outlined">
        <Tabs
          value={tab}
          onChange={(_event, value: NationalTeamsTab) => {
            const next = new URLSearchParams(searchParams)
            next.set('tab', value)
            setSearchParams(next, { replace: true })
          }}
          variant="fullWidth"
          sx={{ borderBottom: 1, borderColor: 'divider' }}
        >
          <Tab value="round" label={t('nationalTeams.tabs.round')} />
          <Tab value="results" label={t('nationalTeams.tabs.results')} />
          <Tab value="world-cup" label={t('nationalTeams.tabs.worldCup')} />
        </Tabs>
        <Box sx={{ p: { xs: 1, sm: 2 } }}>
          {tab === 'round' && <RoundTab />}
          {tab === 'results' && <ResultsTab />}
          {tab === 'world-cup' && <WorldCupTab />}
        </Box>
      </Paper>
    </Stack>
  )
}
