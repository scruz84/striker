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
import { useLineup, useMe, useSetLineup, useUserTeam } from '../../api/hooks'
import { ALL_POSITIONS, POSITION_LABELS } from '../../api/types'
import type { Lineup, Player, Team } from '../../api/types'

type Selections = Record<number, number | ''>

function initialSelections(lineup: Lineup): Selections {
  const next: Selections = {}
  for (const position of ALL_POSITIONS) {
    const existing = lineup.positions.find((p) => p.position === position)
    next[position] = existing ? existing.playerId : ''
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
  // Lazy initializer runs once on mount, seeded from the lineup already loaded by the parent -
  // no effect needed to sync it in after the fact.
  const [selections, setSelections] = useState<Selections>(() => initialSelections(lineup))
  const setLineup = useSetLineup()

  const usedPlayerIds = useMemo(() => new Set(Object.values(selections).filter((v) => v !== '')), [selections])

  const isComplete = ALL_POSITIONS.every((p) => selections[p] !== '' && selections[p] !== undefined)
  const selectedIds = Object.values(selections).filter((v) => v !== '')
  const hasDuplicates = new Set(selectedIds).size !== selectedIds.length

  const handleChange = (position: number) => (event: SelectChangeEvent<number | ''>) => {
    const value = event.target.value
    setSelections((prev) => ({ ...prev, [position]: value === '' ? '' : Number(value) }))
  }

  const handleSubmit = () => {
    if (!isComplete || hasDuplicates) {
      return
    }
    setLineup.mutate(ALL_POSITIONS.map((position) => ({ position, playerId: selections[position] as number })))
  }

  return (
    <Box>
      <Typography variant="h5" gutterBottom>
        Lineup - {team.name}
      </Typography>

      {hasDuplicates && (
        <Alert severity="warning" sx={{ mb: 2 }}>
          Each player can only be selected once.
        </Alert>
      )}
      {setLineup.isError && (
        <Alert severity="error" sx={{ mb: 2 }}>
          Could not save the lineup. Check your selections.
        </Alert>
      )}
      {setLineup.isSuccess && (
        <Alert severity="success" sx={{ mb: 2 }}>
          Lineup saved.
        </Alert>
      )}

      <Grid container spacing={2}>
        {ALL_POSITIONS.map((position) => (
          <Grid size={{ xs: 12, sm: 6 }} key={position}>
            <FormControl fullWidth>
              <InputLabel id={`position-${position}-label`}>{POSITION_LABELS[position]}</InputLabel>
              <Select
                labelId={`position-${position}-label`}
                label={POSITION_LABELS[position]}
                value={selections[position] ?? ''}
                onChange={handleChange(position)}
              >
                <MenuItem value="">
                  <em>None</em>
                </MenuItem>
                {players.map((player) => (
                  <MenuItem
                    key={player.id}
                    value={player.id}
                    disabled={usedPlayerIds.has(player.id) && selections[position] !== player.id}
                  >
                    {player.firstName} {player.lastName} ({player.age})
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Grid>
        ))}
      </Grid>

      <Button
        variant="contained"
        sx={{ mt: 3 }}
        disabled={!isComplete || hasDuplicates || setLineup.isPending}
        onClick={handleSubmit}
      >
        Save lineup
      </Button>
    </Box>
  )
}
