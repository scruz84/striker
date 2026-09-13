import { type SyntheticEvent, useState } from 'react'
import Box from '@mui/material/Box'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Tab from '@mui/material/Tab'
import Tabs from '@mui/material/Tabs'
import Typography from '@mui/material/Typography'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useCupTieDetails, useCupTieLineup, useMe, useSetCupTieLineup, useUserTeam } from '../../api/hooks'
import type { CupTieDetails, Player, Team } from '../../api/types'
import { MatchLineupForm } from '../lineup/MatchLineupForm'
import { Events, Scoreboard } from '../match/MatchScoreboard'
import { Lineups } from '../match/MatchDetailsPage'
import { formatDateTime } from '../../utils/formatDate'

export function CupTieDetailsPage() {
  const { id } = useParams<{ id: string }>()
  const { data: tie, isLoading } = useCupTieDetails(id)
  const { data: loggedUser } = useMe()
  const { data: userTeam } = useUserTeam(!!loggedUser)

  if (isLoading || !tie) {
    return <Skeleton variant="rectangular" height={200} />
  }

  if (tie.homeTeamId === null || tie.awayTeamId === null) {
    return <ByeTie tie={tie} />
  }

  if (!tie.played) {
    const isParticipant = userTeam?.team.id === tie.homeTeamId || userTeam?.team.id === tie.awayTeamId
    return (
      <UnplayedTie
        id={id}
        tie={tie}
        team={isParticipant ? userTeam?.team : undefined}
        players={isParticipant ? userTeam?.players : undefined}
      />
    )
  }

  return <PlayedTie tie={tie} />
}

function ByeTie({ tie }: { tie: CupTieDetails }) {
  const { t } = useTranslation()
  return (
    <Paper variant="outlined" sx={{ p: 3, textAlign: 'center' }}>
      <Typography>{t('cup.bye')}</Typography>
      <Typography color="text.secondary">{tie.homeTeamName || tie.awayTeamName}</Typography>
    </Paper>
  )
}

function UnplayedTie({
  id,
  tie,
  team,
  players,
}: {
  id: string | undefined
  tie: CupTieDetails
  team?: Team
  players?: Player[]
}) {
  return (
    <Box>
      <Scoreboard
        homeId={tie.homeTeamId ?? 0}
        homeName={tie.homeTeamName}
        awayId={tie.awayTeamId ?? 0}
        awayName={tie.awayTeamName}
        homeGoals={0}
        awayGoals={0}
        status="scheduled"
        subtitle={tie.matchDate ? formatDateTime(tie.matchDate) : undefined}
      />
      {id && team && players && <CupTieLineupSection tieId={id} team={team} players={players} />}
    </Box>
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
      players={players}
      lineup={lineup}
      onSave={(formation, positions) => setLineup.mutate({ formation, positions })}
      isSaving={setLineup.isPending}
      isError={setLineup.isError}
      isSuccess={setLineup.isSuccess}
    />
  )
}

function PlayedTie({ tie }: { tie: CupTieDetails }) {
  const { t } = useTranslation()
  const [tab, setTab] = useState(0)
  const handleChange = (_event: SyntheticEvent, newValue: number) => setTab(newValue)

  return (
    <Box>
      <Scoreboard
        homeId={tie.homeTeamId ?? 0}
        homeName={tie.homeTeamName}
        awayId={tie.awayTeamId ?? 0}
        awayName={tie.awayTeamName}
        homeGoals={tie.homeGoals}
        awayGoals={tie.awayGoals}
        status="played"
        subtitle={tie.matchDate ? formatDateTime(tie.matchDate) : undefined}
      />
      {tie.wentToPenalties && (
        <Typography color="text.secondary" align="center" sx={{ mb: 2 }}>
          {t('cup.pens')}
        </Typography>
      )}

      <Paper variant="outlined" sx={{ mb: 2 }}>
        <Tabs value={tab} onChange={handleChange}>
          <Tab label={t('match.tabs.events')} />
          <Tab label={t('match.tabs.lineups')} />
        </Tabs>
        <Box sx={{ p: 2 }}>
          {tab === 0 && (
            <Events events={tie.events} homeId={tie.homeTeamId ?? 0} homeName={tie.homeTeamName} awayName={tie.awayTeamName} />
          )}
          {tab === 1 && (
            <Lineups
              homeName={tie.homeTeamName}
              awayName={tie.awayTeamName}
              homeLineup={tie.homeLineup}
              awayLineup={tie.awayLineup}
              homePlayers={tie.homePlayers}
              awayPlayers={tie.awayPlayers}
              matchInstants={null}
            />
          )}
        </Box>
      </Paper>
    </Box>
  )
}
