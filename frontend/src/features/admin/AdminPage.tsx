import Box from '@mui/material/Box'
import Skeleton from '@mui/material/Skeleton'
import Tab from '@mui/material/Tab'
import Tabs from '@mui/material/Tabs'
import Typography from '@mui/material/Typography'
import DashboardOutlinedIcon from '@mui/icons-material/DashboardOutlined'
import PeopleOutlinedIcon from '@mui/icons-material/PeopleOutlined'
import PublicOutlinedIcon from '@mui/icons-material/PublicOutlined'
import SportsSoccerOutlinedIcon from '@mui/icons-material/SportsSoccerOutlined'
import TuneOutlinedIcon from '@mui/icons-material/TuneOutlined'
import { useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useMe } from '../../api/hooks'
import { DEFAULT_ADMIN_TAB, adminTabFromParam, type AdminTab } from './adminTabs'
import { OverviewTab } from './OverviewTab'
import { RulesTab } from './RulesTab'
import { TeamsTab } from './TeamsTab'
import { UsersTab } from './UsersTab'
import { WorldTab } from './WorldTab'

// The admin panel is split by task into tabs, each of which loads only its own data (every tab
// component runs its own queries, and only the active one is mounted). The active tab lives in the
// URL (?tab=) so a tab can be linked to and survives navigating away and back; writes replace the
// current history entry so switching tabs doesn't pile up "back" stops.
export function AdminPage() {
  const { t } = useTranslation()
  const { data: loggedUser, isLoading: meLoading } = useMe()
  const [searchParams, setSearchParams] = useSearchParams()
  const tab = adminTabFromParam(searchParams.get('tab'))

  if (meLoading) {
    return <Skeleton variant="rectangular" height={300} />
  }

  if (!loggedUser?.isAdmin) {
    return <Typography color="text.secondary">{t('admin.noAccess')}</Typography>
  }

  const handleChange = (_event: unknown, next: AdminTab) => {
    setSearchParams(
      (prev) => {
        const params = new URLSearchParams(prev)
        if (next === DEFAULT_ADMIN_TAB) {
          params.delete('tab')
        } else {
          params.set('tab', next)
        }
        return params
      },
      { replace: true },
    )
  }

  return (
    <Box>
      <Typography variant="h5" gutterBottom>
        {t('admin.title')}
      </Typography>

      <Tabs
        value={tab}
        onChange={handleChange}
        variant="scrollable"
        scrollButtons="auto"
        allowScrollButtonsMobile
        sx={{
          mb: 2,
          borderBottom: 1,
          borderColor: 'divider',
          '& .MuiTab-root': { minWidth: 'auto', minHeight: 48, px: { xs: 1.25, sm: 2 }, fontSize: { xs: '0.8rem', sm: '0.875rem' } },
        }}
      >
        <Tab value="overview" icon={<DashboardOutlinedIcon fontSize="small" />} iconPosition="start" label={t('admin.tabs.overview')} />
        <Tab value="teams" icon={<SportsSoccerOutlinedIcon fontSize="small" />} iconPosition="start" label={t('admin.tabs.teams')} />
        <Tab value="rules" icon={<TuneOutlinedIcon fontSize="small" />} iconPosition="start" label={t('admin.tabs.rules')} />
        <Tab value="world" icon={<PublicOutlinedIcon fontSize="small" />} iconPosition="start" label={t('admin.tabs.world')} />
        <Tab value="users" icon={<PeopleOutlinedIcon fontSize="small" />} iconPosition="start" label={t('admin.tabs.users')} />
      </Tabs>

      {tab === 'overview' && <OverviewTab />}
      {tab === 'teams' && <TeamsTab />}
      {tab === 'rules' && <RulesTab />}
      {tab === 'world' && <WorldTab />}
      {tab === 'users' && <UsersTab />}
    </Box>
  )
}
