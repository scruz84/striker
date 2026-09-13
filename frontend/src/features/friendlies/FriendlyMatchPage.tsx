import { useEffect, useMemo, useState } from 'react'
import {
  closestCenter,
  DndContext,
  DragOverlay,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
} from '@dnd-kit/core'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import FormControl from '@mui/material/FormControl'
import Grid from '@mui/material/Grid'
import InputLabel from '@mui/material/InputLabel'
import MenuItem from '@mui/material/MenuItem'
import Paper from '@mui/material/Paper'
import Select, { type SelectChangeEvent } from '@mui/material/Select'
import Skeleton from '@mui/material/Skeleton'
import Typography from '@mui/material/Typography'
import { useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  queryKeys,
  useFriendlyLineup,
  useFriendlyMatch,
  useMe,
  useSetFriendlyLineup,
  useUserTeam,
} from '../../api/hooks'
import {
  FORMATION_BALANCED,
  FORMATION_LABEL_KEYS,
  FORMATION_POSITION_COUNTS,
  type FriendlyMatch,
  type Lineup,
  type Player,
  type Team,
} from '../../api/types'
import { useLiveMatch } from '../../api/useLiveMatch'
import { CartoonAvatar } from '../../components/avatar/CartoonAvatar'
import { PlayerBench } from '../lineup/PlayerBench'
import { SoccerPitch } from '../lineup/SoccerPitch'
import { Events, Scoreboard } from '../match/MatchScoreboard'
import { formatDateTime } from '../../utils/formatDate'

export function FriendlyMatchPage() {
  const { id } = useParams<{ id: string }>()
  const { data: match, isLoading } = useFriendlyMatch(id)
  const { data: loggedUser } = useMe()
  const { data: userTeam } = useUserTeam(!!loggedUser)

  if (isLoading || !match) {
    return <Skeleton variant="rectangular" height={200} />
  }

  if (!match.played) {
    return <LiveFriendly id={id} match={match} team={userTeam?.team} players={userTeam?.players} />
  }

  return (
    <Scoreboard
      homeId={match.homeTeamId}
      homeName={match.homeTeamName}
      awayId={match.awayTeamId}
      awayName={match.awayTeamName}
      homeGoals={match.homeGoals}
      awayGoals={match.awayGoals}
      status="played"
      subtitle={formatDateTime(match.matchDate)}
    />
  )
}

function LiveFriendly({
  id,
  match,
  team,
  players,
}: {
  id: string | undefined
  match: FriendlyMatch
  team?: Team
  players?: Player[]
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const live = useLiveMatch(id ? `/v1/friendlies/${id}/live` : undefined, !!id)

  useEffect(() => {
    if (live.complete && id) {
      void queryClient.invalidateQueries({ queryKey: queryKeys.friendlyMatch(id) })
    }
  }, [live.complete, id, queryClient])

  if (live.complete) {
    return <Typography color="text.secondary">{t('match.matchFinished')}</Typography>
  }

  return (
    <Box>
      <Scoreboard
        homeId={match.homeTeamId}
        homeName={match.homeTeamName}
        awayId={match.awayTeamId}
        awayName={match.awayTeamName}
        homeGoals={live.homeGoals}
        awayGoals={live.awayGoals}
        status={live.connected ? 'live' : 'connecting'}
        minute={live.minute}
        subtitle={formatDateTime(match.matchDate)}
      />
      <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
        <Events events={live.events} homeId={match.homeTeamId} homeName={match.homeTeamName} awayName={match.awayTeamName} />
      </Paper>
      {id && team && players && <FriendlyLineupEditor matchId={id} team={team} players={players} />}
    </Box>
  )
}

function FriendlyLineupEditor({ matchId, team, players }: { matchId: string; team: Team; players: Player[] }) {
  const { data: lineup, isLoading } = useFriendlyLineup(matchId, true)
  if (isLoading || !lineup) {
    return <Skeleton variant="rectangular" height={300} />
  }
  return <FriendlyLineupForm matchId={matchId} team={team} players={players} lineup={lineup} />
}

// Slots per position, keyed the same way FORMATION_POSITION_COUNTS is - see
// LineupEditorPage.tsx's own copy of this shape for the default-lineup editor.
type Selections = Record<number, (number | '')[]>

function initialSelections(lineup: Lineup): Selections {
  const next: Selections = {}
  for (const pos of lineup.positions) {
    if (!next[pos.position]) {
      next[pos.position] = []
    }
    next[pos.position].push(pos.playerId)
  }
  return next
}

// A trimmed copy of LineupEditorPage's formation/pitch/bench editor, scoped to one friendly
// instead of the team's standing default lineup - saved via useSetFriendlyLineup instead of
// useSetLineup. Deliberately has no scheduled-substitutions section: that's a default-lineup-only
// concept the friendly override doesn't need (see backend common.SetFriendlyLineup).
function FriendlyLineupForm({ matchId, team, players, lineup }: { matchId: string; team: Team; players: Player[]; lineup: Lineup }) {
  const { t } = useTranslation()
  const [formation, setFormation] = useState(lineup.formation)
  const [selections, setSelections] = useState<Selections>(() => initialSelections(lineup))
  const [activePlayerId, setActivePlayerId] = useState<number | null>(null)
  const setLineup = useSetFriendlyLineup(matchId)

  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 4 } }), useSensor(KeyboardSensor))

  const counts = FORMATION_POSITION_COUNTS[formation] ?? FORMATION_POSITION_COUNTS[FORMATION_BALANCED]
  const positionsToShow = Object.keys(counts).map(Number)

  const playersById = useMemo(() => new Map(players.map((p) => [p.id, p])), [players])

  const visibleSelections = useMemo(() => {
    const flat: { position: number; playerId: number | '' }[] = []
    for (const position of positionsToShow) {
      const count = counts[position] ?? 0
      for (let i = 0; i < count; i++) {
        flat.push({ position, playerId: selections[position]?.[i] ?? '' })
      }
    }
    return flat
  }, [selections, counts, positionsToShow])

  const usedPlayerIds = useMemo(
    () => new Set(visibleSelections.map((s) => s.playerId).filter((v) => v !== '')),
    [visibleSelections],
  )
  const isComplete = visibleSelections.every((s) => s.playerId !== '')
  const benchPlayers = players.filter((p) => !usedPlayerIds.has(p.id))

  const getPlayer = (position: number, slotIndex: number): Player | undefined => {
    const id = selections[position]?.[slotIndex]
    return id === undefined || id === '' ? undefined : playersById.get(id)
  }

  function findSlotForPlayer(playerId: number): { position: number; slotIndex: number } | null {
    for (const position of positionsToShow) {
      const count = counts[position] ?? 0
      const arr = selections[position] ?? []
      for (let i = 0; i < count; i++) {
        if (arr[i] === playerId) {
          return { position, slotIndex: i }
        }
      }
    }
    return null
  }

  const handleFormationChange = (event: SelectChangeEvent<number>) => {
    setFormation(Number(event.target.value))
  }

  const handleDragStart = (event: DragStartEvent) => {
    const playerId = event.active.data.current?.playerId as number | undefined
    setActivePlayerId(playerId ?? null)
  }

  const handleDragEnd = (event: DragEndEvent) => {
    setActivePlayerId(null)
    const { active, over } = event
    if (!over) return
    const playerId = active.data.current?.playerId as number | undefined
    if (playerId === undefined) return

    const source = findSlotForPlayer(playerId)

    if (over.id === 'bench') {
      if (!source) return
      setSelections((prev) => {
        const arr = [...(prev[source.position] ?? [])]
        arr[source.slotIndex] = ''
        return { ...prev, [source.position]: arr }
      })
      return
    }

    const targetPosition = over.data.current?.position as number | undefined
    const targetSlotIndex = over.data.current?.slotIndex as number | undefined
    if (targetPosition === undefined || targetSlotIndex === undefined) return
    if (source && source.position === targetPosition && source.slotIndex === targetSlotIndex) return

    setSelections((prev) => {
      const next = { ...prev }
      const targetArr = [...(next[targetPosition] ?? [])]
      const displaced = targetArr[targetSlotIndex] ?? ''
      targetArr[targetSlotIndex] = playerId
      next[targetPosition] = targetArr

      if (source) {
        const sourceArr = source.position === targetPosition ? targetArr : [...(next[source.position] ?? [])]
        sourceArr[source.slotIndex] = displaced
        next[source.position] = sourceArr
      }
      return next
    })
  }

  const handleSubmit = () => {
    if (!isComplete) return
    const positions = visibleSelections.map((s) => ({ position: s.position, playerId: s.playerId as number }))
    setLineup.mutate({ formation, positions })
  }

  return (
    <Paper variant="outlined" sx={{ p: 2 }}>
      <Typography variant="h6" gutterBottom>
        {t('friendlies.lineup.title', { teamName: team.name })}
      </Typography>

      <FormControl sx={{ mb: 2, minWidth: 220 }}>
        <InputLabel id="friendly-formation-label">{t('lineup.formation')}</InputLabel>
        <Select
          labelId="friendly-formation-label"
          label={t('lineup.formation')}
          value={formation}
          onChange={handleFormationChange}
        >
          {Object.keys(FORMATION_LABEL_KEYS).map((value) => (
            <MenuItem key={value} value={Number(value)}>
              {t(FORMATION_LABEL_KEYS[Number(value)])}
            </MenuItem>
          ))}
        </Select>
      </FormControl>

      {!isComplete && (
        <Alert severity="info" sx={{ mb: 2 }}>
          {t('lineup.incompleteWarning')}
        </Alert>
      )}
      {setLineup.isError && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {t('lineup.saveError')}
        </Alert>
      )}
      {setLineup.isSuccess && (
        <Alert severity="success" sx={{ mb: 2 }}>
          {t('lineup.saveSuccess')}
        </Alert>
      )}

      <DndContext sensors={sensors} collisionDetection={closestCenter} onDragStart={handleDragStart} onDragEnd={handleDragEnd}>
        <Grid container spacing={2}>
          <Grid size={{ xs: 12, md: 8 }}>
            <SoccerPitch counts={counts} getPlayer={getPlayer} />
          </Grid>
          <Grid size={{ xs: 12, md: 4 }}>
            <PlayerBench players={benchPlayers} />
          </Grid>
        </Grid>
        <DragOverlay dropAnimation={null}>
          {activePlayerId !== null && (
            <Box sx={{ filter: 'drop-shadow(0 2px 4px rgba(0,0,0,0.4))', cursor: 'grabbing' }}>
              <CartoonAvatar seed={activePlayerId} size={48} />
            </Box>
          )}
        </DragOverlay>
      </DndContext>

      <Button variant="contained" sx={{ mt: 3 }} disabled={!isComplete || setLineup.isPending} onClick={handleSubmit}>
        {t('lineup.save')}
      </Button>
    </Paper>
  )
}
