import { useMemo, useState } from 'react'
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
import Select, { type SelectChangeEvent } from '@mui/material/Select'
import Skeleton from '@mui/material/Skeleton'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useLineup, useMe, useSetLineup, useUserTeam } from '../../api/hooks'
import { FORMATION_BALANCED, FORMATION_LABEL_KEYS, FORMATION_POSITION_COUNTS } from '../../api/types'
import type { Lineup, Player, Team } from '../../api/types'
import { CartoonAvatar } from '../../components/avatar/CartoonAvatar'
import { PlayerBench } from './PlayerBench'
import { SoccerPitch } from './SoccerPitch'

// Slots per position, keyed the same way FORMATION_POSITION_COUNTS is: a position not present
// for a formation simply isn't used by it.
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

export function LineupEditorPage() {
  const { data: loggedUser } = useMe()
  const { data: userTeam, isLoading: teamLoading } = useUserTeam(!!loggedUser)
  const { data: lineup, isLoading: lineupLoading } = useLineup(!!loggedUser)

  if (teamLoading || lineupLoading || !userTeam || !lineup) {
    return <Skeleton variant="rectangular" height={300} />
  }

  return <LineupForm team={userTeam.team} players={userTeam.players} lineup={lineup} />
}

function LineupForm({ team, players, lineup }: { team: Team; players: Player[]; lineup: Lineup }) {
  const { t } = useTranslation()
  // Lazy initializers run once on mount, seeded from the lineup already loaded by the parent -
  // no effect needed to sync them in after the fact. Switching formations doesn't reset
  // selections: it just changes how many slots per position are shown/required, so a player
  // picked under one formation stays picked (just off-pitch, back on the bench) if you switch
  // away and later switch back to it.
  const [formation, setFormation] = useState(lineup.formation)
  const [selections, setSelections] = useState<Selections>(() => initialSelections(lineup))
  const [activePlayerId, setActivePlayerId] = useState<number | null>(null)
  const setLineup = useSetLineup()

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
    if (!isComplete) {
      return
    }
    const positions = visibleSelections.map((s) => ({ position: s.position, playerId: s.playerId as number }))
    setLineup.mutate({ formation, positions })
  }

  return (
    <Box>
      <Typography variant="h5" gutterBottom>
        {t('lineup.title', { teamName: team.name })}
      </Typography>

      <FormControl sx={{ mb: 2, minWidth: 220 }}>
        <InputLabel id="formation-label">{t('lineup.formation')}</InputLabel>
        <Select labelId="formation-label" label={t('lineup.formation')} value={formation} onChange={handleFormationChange}>
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

      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragStart={handleDragStart}
        onDragEnd={handleDragEnd}
      >
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

      <Button
        variant="contained"
        sx={{ mt: 3 }}
        disabled={!isComplete || setLineup.isPending}
        onClick={handleSubmit}
      >
        {t('lineup.save')}
      </Button>
    </Box>
  )
}
