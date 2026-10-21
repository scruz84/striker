import { useState } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Skeleton from '@mui/material/Skeleton'
import Tab from '@mui/material/Tab'
import Tabs from '@mui/material/Tabs'
import { useTranslation } from 'react-i18next'
import {
  useFriendlyDefaultLineup,
  useLineup,
  useMe,
  useSetFriendlyDefaultLineup,
  useSetLineup,
  useTeamMatches,
  useUserTeam,
} from '../../api/hooks'
import type { Lineup, LineupPosition, LineupSubstitution, Player, Team } from '../../api/types'
import { isPast } from '../../api/useLiveMatch'
import { PageHeader } from '../../components/PageHeader'
import { formatDayTime } from '../../utils/formatDate'
import { MatchLineupForm } from './MatchLineupForm'

export function LineupEditorPage() {
  const { t } = useTranslation()
  const [tab, setTab] = useState(0)
  const { data: loggedUser } = useMe()
  const { data: userTeam, isLoading: teamLoading } = useUserTeam(!!loggedUser)
  const { data: lineup, isLoading: lineupLoading } = useLineup(!!loggedUser && tab === 0)
  const { data: friendlyLineup, isLoading: friendlyLineupLoading } = useFriendlyDefaultLineup(!!loggedUser && tab === 1)

  const { data: teamMatches } = useTeamMatches(userTeam ? String(userTeam.team.id) : undefined)

  if (teamLoading || !userTeam) {
    return <Skeleton variant="rectangular" height={300} />
  }

  // "Contra quién y cuándo" is the useful context for a lineup - the club is already in the
  // header, so it no longer repeats in the title.
  const nextMatch = teamMatches?.filter((m) => !isPast(m.matchDate)).sort((a, b) => a.matchDate.localeCompare(b.matchDate))[0]
  const subtitle = nextMatch
    ? t('lineup.subtitle', {
        opponent: nextMatch.opponentName,
        date: formatDayTime(nextMatch.matchDate),
        side: t(nextMatch.isHome ? 'common.home' : 'common.away'),
      })
    : t('lineup.subtitleNone')

  return (
    <Box>
      <PageHeader title={t('header.nav.lineup')} subtitle={subtitle} />
      <Tabs value={tab} onChange={(_, next: number) => setTab(next)} variant="fullWidth" sx={{ mb: { xs: 1, sm: 2 } }}>
        <Tab label={t('lineup.tabs.leagueCup')} />
        <Tab label={t('lineup.tabs.friendly')} />
      </Tabs>
      {tab === 0 &&
        (lineupLoading || !lineup ? (
          <Skeleton variant="rectangular" height={300} />
        ) : (
          <LineupForm team={userTeam.team} players={userTeam.players} lineup={lineup} />
        ))}
      {tab === 1 &&
        (friendlyLineupLoading || !friendlyLineup ? (
          <Skeleton variant="rectangular" height={300} />
        ) : (
          <FriendlyDefaultLineupForm team={userTeam.team} players={userTeam.players} lineup={friendlyLineup} />
        ))}
    </Box>
  )
}

// The friendly-only default lineup, used as the fallback for any friendly without a per-match
// override (see backend common.GetLineupsForFriendly) - independent of the league/cup default
// above, so a manager can field a weaker/experimental XI in friendlies without touching their
// competitive default. Reuses MatchLineupForm (no scheduled-substitutions section: those stay
// team-wide, shared with the league/cup default).
function FriendlyDefaultLineupForm({ team, players, lineup }: { team: Team; players: Player[]; lineup: Lineup }) {
  const { t } = useTranslation()
  const setFriendlyLineup = useSetFriendlyDefaultLineup()

  return (
    <Box>
      <Alert severity="info" sx={{ mb: { xs: 1, sm: 2 } }}>
        {t('lineup.friendlyHint')}
      </Alert>
      <MatchLineupForm
        titleKey="lineup.title"
        titleValues={{ teamName: team.name }}
        teamId={team.id}
        players={players}
        lineup={lineup}
        onSave={(formation: number, positions: LineupPosition[]) => setFriendlyLineup.mutate({ formation, positions })}
        isSaving={setFriendlyLineup.isPending}
        isError={setFriendlyLineup.isError}
        isSuccess={setFriendlyLineup.isSuccess}
        isDefaultLineup
        embedded
      />
    </Box>
  )
}

function LineupForm({ team, players, lineup }: { team: Team; players: Player[]; lineup: Lineup }) {
  const setLineup = useSetLineup()

  return (
    <MatchLineupForm
      titleKey="lineup.title"
      titleValues={{ teamName: team.name }}
      teamId={team.id}
      players={players}
      lineup={lineup}
      onSave={(formation: number, positions: LineupPosition[], substitutions: LineupSubstitution[]) =>
        setLineup.mutate({ formation, positions, substitutions })
      }
      isSaving={setLineup.isPending}
      isError={setLineup.isError}
      isSuccess={setLineup.isSuccess}
      substitutionsEnabled
      isDefaultLineup
      embedded
    />
  )
}
