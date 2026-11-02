import Skeleton from '@mui/material/Skeleton'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useFriendlyLineup, useFriendlyMatch, useMe, useSetFriendlyLineup, useUserTeam } from '../../api/hooks'
import { MATCH_KIND_FRIENDLY, type Player, type Team } from '../../api/types'
import { MatchLineupForm } from '../lineup/MatchLineupForm'
import { OpponentPreviewCard } from '../lineup/OpponentPreviewCard'
import { MatchViewer } from '../match/MatchViewer'
import { fromFriendly } from '../match/matchViewModel'

// The friendly route (/friendlies/:id, which also serves national-team friendlies and World Cup group
// matches): loads the friendly, maps it onto the shared MatchView and hands it to MatchViewer. The
// lineup editor is built only when the logged-in owner's own team plays the match - it used to be
// offered to any owner.
export function FriendlyMatchPage() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const { data: match, isLoading } = useFriendlyMatch(id)
  const { data: loggedUser } = useMe()
  const { data: userTeam } = useUserTeam(!!loggedUser)

  if (isLoading || !match) {
    return <Skeleton variant="rectangular" height={200} />
  }

  const isParticipant = userTeam?.team.id === match.homeTeamId || userTeam?.team.id === match.awayTeamId
  return (
    <MatchViewer
      key={id}
      view={fromFriendly(match, t)}
      lineupSection={id && isParticipant && userTeam ? <FriendlyLineupEditor matchId={id} team={userTeam.team} players={userTeam.players} /> : undefined}
      previewSection={
        id && isParticipant && userTeam ? (
          <OpponentPreviewCard kind={MATCH_KIND_FRIENDLY} matchId={id} teamId={userTeam.team.id} teamName={userTeam.team.name} />
        ) : undefined
      }
    />
  )
}

function FriendlyLineupEditor({ matchId, team, players }: { matchId: string; team: Team; players: Player[] }) {
  const { data: lineup, isLoading } = useFriendlyLineup(matchId, true)
  const setLineup = useSetFriendlyLineup(matchId)
  if (isLoading || !lineup) {
    return <Skeleton variant="rectangular" height={300} />
  }
  return (
    <MatchLineupForm
      titleKey="friendlies.lineup.title"
      titleValues={{ teamName: team.name }}
      teamId={team.id}
      players={players}
      lineup={lineup}
      onSave={(formation, positions, substitutions) => setLineup.mutate({ formation, positions, substitutions })}
      isSaving={setLineup.isPending}
      isError={setLineup.isError}
      isSuccess={setLineup.isSuccess}
      substitutionsEnabled
      isFriendly
    />
  )
}
