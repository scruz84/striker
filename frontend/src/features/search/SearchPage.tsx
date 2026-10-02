import { useState } from 'react'
import ApartmentTwoToneIcon from '@mui/icons-material/ApartmentTwoTone'
import GroupsTwoToneIcon from '@mui/icons-material/GroupsTwoTone'
import PersonTwoToneIcon from '@mui/icons-material/PersonTwoTone'
import SearchTwoToneIcon from '@mui/icons-material/SearchTwoTone'
import SportsTwoToneIcon from '@mui/icons-material/SportsTwoTone'
import Divider from '@mui/material/Divider'
import InputAdornment from '@mui/material/InputAdornment'
import List from '@mui/material/List'
import ListItemButton from '@mui/material/ListItemButton'
import ListItemIcon from '@mui/material/ListItemIcon'
import ListItemText from '@mui/material/ListItemText'
import ListSubheader from '@mui/material/ListSubheader'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useSearch } from '../../api/hooks'
import type { CoachSearchResult, PlayerSearchResult, StadiumSearchResult, TeamSearchResult } from '../../api/types'

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
      <Typography variant="h5" sx={{ fontWeight: 700 }}>
        {t('search.title')}
      </Typography>
      <Typography variant="body2" color="text.secondary">
        {t('search.subtitle')}
      </Typography>

      <TextField
        autoFocus
        fullWidth
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        placeholder={t('search.placeholder')}
        slotProps={{
          input: {
            startAdornment: (
              <InputAdornment position="start">
                <SearchTwoToneIcon color="action" />
              </InputAdornment>
            ),
          },
        }}
      />

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
              icon={<GroupsTwoToneIcon />}
              items={results.teams}
              renderItem={(team) => <TeamResultRow key={team.id} team={team} />}
            />
            <ResultSection
              title={t('search.players')}
              icon={<PersonTwoToneIcon />}
              items={results.players}
              renderItem={(player) => <PlayerResultRow key={player.id} player={player} />}
            />
            <ResultSection
              title={t('search.coaches')}
              icon={<SportsTwoToneIcon />}
              items={results.coaches}
              renderItem={(coach) => <CoachResultRow key={coach.id} coach={coach} />}
            />
            <ResultSection
              title={t('search.stadiums')}
              icon={<ApartmentTwoToneIcon />}
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
        <GroupsTwoToneIcon />
      </ListItemIcon>
      <ListItemText primary={team.name} secondary={team.divisionName || undefined} />
    </ListItemButton>
  )
}

function PlayerResultRow({ player }: { player: PlayerSearchResult }) {
  const { t } = useTranslation()
  return (
    <ListItemButton component={RouterLink} to={`/player/${player.id}`}>
      <ListItemIcon>
        <PersonTwoToneIcon />
      </ListItemIcon>
      <ListItemText
        primary={`${player.firstName} ${player.lastName}`}
        secondary={player.teamName || t('search.freeAgent')}
      />
    </ListItemButton>
  )
}

function CoachResultRow({ coach }: { coach: CoachSearchResult }) {
  const { t } = useTranslation()
  return (
    <ListItemButton component={RouterLink} to={`/team/${coach.teamId}`}>
      <ListItemIcon>
        <SportsTwoToneIcon />
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
        <ApartmentTwoToneIcon />
      </ListItemIcon>
      <ListItemText primary={stadium.stadiumName} secondary={t('search.stadiumOf', { team: stadium.teamName })} />
    </ListItemButton>
  )
}
