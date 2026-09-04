import { useMemo, useState } from 'react'
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
import { FORMATION_BALANCED, FORMATION_LABEL_KEYS, FORMATION_POSITION_COUNTS, POSITION_LABEL_KEYS } from '../../api/types'
import type { Lineup, Player, Team } from '../../api/types'

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
  // picked under one formation stays picked if you switch back to it.
  const [formation, setFormation] = useState(lineup.formation)
  const [selections, setSelections] = useState<Selections>(() => initialSelections(lineup))
  const setLineup = useSetLineup()

  const counts = FORMATION_POSITION_COUNTS[formation] ?? FORMATION_POSITION_COUNTS[FORMATION_BALANCED]
  const positionsToShow = Object.keys(counts).map(Number)

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
  const selectedIds = visibleSelections.map((s) => s.playerId).filter((v) => v !== '')
  const hasDuplicates = new Set(selectedIds).size !== selectedIds.length

  const handleFormationChange = (event: SelectChangeEvent<number>) => {
    setFormation(Number(event.target.value))
  }

  const handleSlotChange = (position: number, slotIndex: number) => (event: SelectChangeEvent<number | ''>) => {
    const value = event.target.value
    setSelections((prev) => {
      const arr = [...(prev[position] ?? [])]
      arr[slotIndex] = value === '' ? '' : Number(value)
      return { ...prev, [position]: arr }
    })
  }

  const handleSubmit = () => {
    if (!isComplete || hasDuplicates) {
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

      {hasDuplicates && (
        <Alert severity="warning" sx={{ mb: 2 }}>
          {t('lineup.duplicateWarning')}
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

      <Grid container spacing={2}>
        {positionsToShow.map((position) =>
          Array.from({ length: counts[position] ?? 0 }, (_, slotIndex) => {
            const positionLabel = t(POSITION_LABEL_KEYS[position])
            const label =
              (counts[position] ?? 0) > 1
                ? t('lineup.slotLabel', { position: positionLabel, index: slotIndex + 1 })
                : positionLabel
            const value = selections[position]?.[slotIndex] ?? ''
            const labelId = `position-${position}-${slotIndex}-label`
            return (
              <Grid size={{ xs: 12, sm: 6 }} key={`${position}-${slotIndex}`}>
                <FormControl fullWidth>
                  <InputLabel id={labelId}>{label}</InputLabel>
                  <Select labelId={labelId} label={label} value={value} onChange={handleSlotChange(position, slotIndex)}>
                    <MenuItem value="">
                      <em>{t('lineup.none')}</em>
                    </MenuItem>
                    {players.map((player) => (
                      <MenuItem key={player.id} value={player.id} disabled={usedPlayerIds.has(player.id) && value !== player.id}>
                        {player.firstName} {player.lastName} ({player.age})
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              </Grid>
            )
          }),
        )}
      </Grid>

      <Button
        variant="contained"
        sx={{ mt: 3 }}
        disabled={!isComplete || hasDuplicates || setLineup.isPending}
        onClick={handleSubmit}
      >
        {t('lineup.save')}
      </Button>
    </Box>
  )
}
