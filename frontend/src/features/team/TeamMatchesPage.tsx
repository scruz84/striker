import Stack from '@mui/material/Stack'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useTeamDetails } from '../../api/hooks'
import { TeamMatchesSection } from './TeamMatchesSection'
import { PageHeader } from '../../components/PageHeader'

// A dedicated page for a team's match history/schedule, reachable from its own nav sidebar entry
// (see NavDrawer) - previously this only lived inline on the general team page, mixed in with
// squad/academy/trophy cabinet content (issue #38).
export function TeamMatchesPage() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const { data: teamDetails, isLoading } = useTeamDetails(id)

  if (isLoading || !teamDetails || !id) {
    return <div>{t('common.loading')}</div>
  }

  return (
    <Stack spacing={1}>
      <PageHeader title={t('team.matches.title')} subtitle={teamDetails.team.name} />
      <TeamMatchesSection teamId={id} />
    </Stack>
  )
}
