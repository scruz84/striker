import Skeleton from '@mui/material/Skeleton'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useDivisionDetails, useMatchDetails, useMatchLineup, useMe, useSetMatchLineup, useUserTeam } from '../../api/hooks'
import { MATCH_KIND_LEAGUE, type Player, type Team } from '../../api/types'
import { MatchLineupForm } from '../lineup/MatchLineupForm'
import { OpponentPreviewCard } from '../lineup/OpponentPreviewCard'
import { MatchViewer } from './MatchViewer'
import { fromWeekMatch } from './matchViewModel'

// Re-exported so existing importers keep working: the helpers live next to the viewer that uses them.
export { Lineups, goalkeeperRating, type LineupsProps } from './Lineups'
export { positionOf } from './matchViewModel'

// The league route (/match/:id): loads the league match and its division, maps them onto the shared
// MatchView and hands them to MatchViewer. The lineup editor is built here, only for an owner whose
// own team plays the match.
export function MatchDetailsPage() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const { data: match, isLoading } = useMatchDetails(id)
  const { data: division } = useDivisionDetails(match ? String(match.divisionId) : undefined)
  const { data: loggedUser } = useMe()
  const { data: userTeam } = useUserTeam(!!loggedUser)

  if (isLoading || !match) {
    return <Skeleton variant="rectangular" height={200} />
  }

  const isParticipant = userTeam?.team.id === match.home || userTeam?.team.id === match.away
  return (
    <MatchViewer
      key={id}
      view={fromWeekMatch(match, division, t)}
      lineupSection={
        id && isParticipant && userTeam ? <MatchLineupSection matchId={id} team={userTeam.team} players={userTeam.players} /> : undefined
      }
      previewSection={
        id && isParticipant && userTeam ? (
          <OpponentPreviewCard kind={MATCH_KIND_LEAGUE} matchId={id} teamId={userTeam.team.id} teamName={userTeam.team.name} />
        ) : undefined
      }
    />
  )
}

function MatchLineupSection({ matchId, team, players }: { matchId: string; team: Team; players: Player[] }) {
  const { data: lineup, isLoading } = useMatchLineup(matchId, true)
  const setLineup = useSetMatchLineup(matchId)
  if (isLoading || !lineup) {
    return <Skeleton variant="rectangular" height={300} />
  }
  return (
    <MatchLineupForm
      titleKey="match.lineup.title"
      titleValues={{ teamName: team.name }}
      teamId={team.id}
      players={players}
      lineup={lineup}
      onSave={(formation, positions, substitutions, mentality) => setLineup.mutate({ formation, positions, substitutions, mentality })}
      isSaving={setLineup.isPending}
      isError={setLineup.isError}
      isSuccess={setLineup.isSuccess}
      substitutionsEnabled
    />
  )
}
