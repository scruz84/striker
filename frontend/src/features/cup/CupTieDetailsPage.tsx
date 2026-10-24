import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Typography from '@mui/material/Typography'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useCupDetails, useCupTieDetails, useCupTieLineup, useMe, useSetCupTieLineup, useUserTeam } from '../../api/hooks'
import type { CupTieDetails, Player, Team } from '../../api/types'
import { localizedTeamName } from '../../utils/nationalTeamName'
import { MatchLineupForm } from '../lineup/MatchLineupForm'
import { MatchViewer } from '../match/MatchViewer'
import { fromCupTie, type MatchView } from '../match/matchViewModel'

// The cup tie route (/cup/ties/:id): loads the tie and its cup (for the round label), maps them onto
// the shared MatchView and hands them to MatchViewer. Cup-only extras are slots of the viewer: the
// penalty-shootout line under a finished scoreboard, and the "bye" card in place of the whole page.
export function CupTieDetailsPage() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const { data: tie, isLoading } = useCupTieDetails(id)
  const { data: cup } = useCupDetails(tie?.cupId)
  const { data: loggedUser } = useMe()
  const { data: userTeam } = useUserTeam(!!loggedUser)

  if (isLoading || !tie) {
    return <Skeleton variant="rectangular" height={200} />
  }

  if (tie.homeTeamId === null || tie.awayTeamId === null) {
    return <ByeTie tie={tie} />
  }

  const view = fromCupTie(tie, cup, t)
  const isParticipant = userTeam?.team.id === tie.homeTeamId || userTeam?.team.id === tie.awayTeamId
  return (
    <MatchViewer
      key={id}
      view={view}
      lineupSection={id && isParticipant && userTeam ? <CupTieLineupSection tieId={id} team={userTeam.team} players={userTeam.players} /> : undefined}
      belowScoreboard={<PenaltyShootoutLine view={view} />}
    />
  )
}

function ByeTie({ tie }: { tie: CupTieDetails }) {
  const { t } = useTranslation()
  return (
    <Paper variant="outlined" sx={{ p: 3, textAlign: 'center' }}>
      <Typography>{t('cup.bye')}</Typography>
      <Typography color="text.secondary">{localizedTeamName(t, tie.homeTeamName || tie.awayTeamName)}</Typography>
    </Paper>
  )
}

// A tie level after 90 minutes was settled on penalties: the line under the finished scoreboard names
// who won the shootout (or just "(pens)" when the winner isn't known).
function PenaltyShootoutLine({ view }: { view: MatchView }) {
  const { t } = useTranslation()
  if (!view.wentToPenalties) {
    return null
  }
  return (
    <Typography color="text.secondary" align="center" sx={{ mb: 2 }}>
      {view.penaltyWinnerName ? t('cup.pensWinner', { team: view.penaltyWinnerName }) : t('cup.pens')}
    </Typography>
  )
}

function CupTieLineupSection({ tieId, team, players }: { tieId: string; team: Team; players: Player[] }) {
  const { data: lineup, isLoading } = useCupTieLineup(tieId, true)
  const setLineup = useSetCupTieLineup(tieId)
  if (isLoading || !lineup) {
    return <Skeleton variant="rectangular" height={300} />
  }
  return (
    <MatchLineupForm
      titleKey="cup.lineup.title"
      titleValues={{ teamName: team.name }}
      teamId={team.id}
      players={players}
      lineup={lineup}
      onSave={(formation, positions, substitutions) => setLineup.mutate({ formation, positions, substitutions })}
      isSaving={setLineup.isPending}
      isError={setLineup.isError}
      isSuccess={setLineup.isSuccess}
      substitutionsEnabled
    />
  )
}
