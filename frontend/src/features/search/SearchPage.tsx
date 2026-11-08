import { useState } from 'react'
import ApartmentOutlinedIcon from '@mui/icons-material/ApartmentOutlined'
import GroupsOutlinedIcon from '@mui/icons-material/GroupsOutlined'
import PersonOutlinedIcon from '@mui/icons-material/PersonOutlined'
import SportsOutlinedIcon from '@mui/icons-material/SportsOutlined'
import Divider from '@mui/material/Divider'
import List from '@mui/material/List'
import ListItem from '@mui/material/ListItem'
import ListItemButton from '@mui/material/ListItemButton'
import ListItemIcon from '@mui/material/ListItemIcon'
import ListItemText from '@mui/material/ListItemText'
import ListSubheader from '@mui/material/ListSubheader'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { SearchBar } from '../../components/m3/SearchBar'
import { WatchToggle } from '../../components/WatchToggle'
import { useSearch } from '../../api/hooks'
import type { CoachSearchResult, PlayerSearchResult, StadiumSearchResult, TeamSearchResult } from '../../api/types'
import { PageHeader } from '../../components/PageHeader'
import { CountryFlag } from '../../components/CountryFlag'

// A single global search page - look up a team, player, coach, or stadium by name and jump
// straight into its details (a coach or stadium has no page of its own, so those results land on
// the team that employs/owns them, same as TeamDetailsPage's CoachCard/StadiumCard).
export function SearchPage() {
  const { t } = useTranslation()
  const [query, setQuery] = useState('')
  const { data: results, isFetching } = useSearch(query)
  const trimmed = query.trim()

  const totalResults = results
    ? results.teams.length + results.players.length + results.coaches.length + results.stadiums.length
    : 0

  return (
    <Stack spacing={2}>
      <PageHeader title={t('search.title')} subtitle={t('search.subtitle')} />

      <SearchBar autoFocus value={query} onChange={setQuery} placeholder={t('search.placeholder')} />

      {!trimmed && (
        <Typography variant="body2" color="text.secondary">
          {t('search.prompt')}
        </Typography>
      )}

      {trimmed && isFetching && !results && <Skeleton variant="rectangular" height={200} sx={{ borderRadius: 2 }} />}

      {trimmed && results && totalResults === 0 && (
        <Typography variant="body2" color="text.secondary">
          {t('search.none', { query: trimmed })}
        </Typography>
      )}

      {trimmed && results && totalResults > 0 && (
        <Paper variant="outlined">
          <List disablePadding>
            <ResultSection
              title={t('search.teams')}
              icon={<GroupsOutlinedIcon />}
              items={results.teams}
              renderItem={(team) => <TeamResultRow key={team.id} team={team} />}
            />
            <ResultSection
              title={t('search.players')}
              icon={<PersonOutlinedIcon />}
              items={results.players}
              renderItem={(player) => <PlayerResultRow key={player.id} player={player} />}
            />
            <ResultSection
              title={t('search.coaches')}
              icon={<SportsOutlinedIcon />}
              items={results.coaches}
              renderItem={(coach) => <CoachResultRow key={coach.id} coach={coach} />}
            />
            <ResultSection
              title={t('search.stadiums')}
              icon={<ApartmentOutlinedIcon />}
              items={results.stadiums}
              renderItem={(stadium) => <StadiumResultRow key={stadium.teamId} stadium={stadium} />}
            />
          </List>
        </Paper>
      )}
    </Stack>
  )
}

function ResultSection<T>({
  title,
  icon,
  items,
  renderItem,
}: {
  title: string
  icon: React.ReactNode
  items: T[]
  renderItem: (item: T) => React.ReactNode
}) {
  if (items.length === 0) {
    return null
  }
  return (
    <>
      <ListSubheader sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        {icon}
        {title}
      </ListSubheader>
      {items.map((item) => renderItem(item))}
      <Divider component="li" />
    </>
  )
}

function TeamResultRow({ team }: { team: TeamSearchResult }) {
  return (
    <ListItemButton component={RouterLink} to={`/team/${team.id}`}>
      <ListItemIcon>
        <GroupsOutlinedIcon />
      </ListItemIcon>
      <ListItemText primary={team.name} secondary={team.divisionName || undefined} />
    </ListItemButton>
  )
}

function PlayerResultRow({ player }: { player: PlayerSearchResult }) {
  const { t } = useTranslation()
  // The follow star sits beside the row's link, not inside it.
  return (
    <ListItem disablePadding secondaryAction={<WatchToggle player={player} />}>
      <ListItemButton component={RouterLink} to={`/player/${player.id}`}>
        <ListItemIcon>
          <PersonOutlinedIcon />
        </ListItemIcon>
        <ListItemText
          primary={
            <>
              <CountryFlag code={player.birthCountry} height={14} marginRight={0.5} />
              {player.firstName} {player.lastName}
            </>
          }
          secondary={player.teamName || t('search.freeAgent')}
        />
      </ListItemButton>
    </ListItem>
  )
}

function CoachResultRow({ coach }: { coach: CoachSearchResult }) {
  const { t } = useTranslation()
  return (
    <ListItemButton component={RouterLink} to={`/team/${coach.teamId}`}>
      <ListItemIcon>
        <SportsOutlinedIcon />
      </ListItemIcon>
      <ListItemText
        primary={`${coach.firstName} ${coach.lastName}`}
        secondary={t('search.coachOf', { team: coach.teamName })}
      />
    </ListItemButton>
  )
}

function StadiumResultRow({ stadium }: { stadium: StadiumSearchResult }) {
  const { t } = useTranslation()
  return (
    <ListItemButton component={RouterLink} to={`/team/${stadium.teamId}`}>
      <ListItemIcon>
        <ApartmentOutlinedIcon />
      </ListItemIcon>
      <ListItemText primary={stadium.stadiumName} secondary={t('search.stadiumOf', { team: stadium.teamName })} />
    </ListItemButton>
  )
}
