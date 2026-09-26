import { useState } from 'react'
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
import Stack from '@mui/material/Stack'
import Tab from '@mui/material/Tab'
import Tabs from '@mui/material/Tabs'
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import {
  useFriendlyDefaultLineup,
  useLineup,
  useMe,
  useSetFriendlyDefaultLineup,
  useSetLineup,
  useUserTeam,
} from '../../api/hooks'
import { FORMATION_LABEL_KEYS, MAX_SCHEDULED_SUBSTITUTIONS } from '../../api/types'
import type { Lineup, LineupPosition, LineupSubstitution, Player, Team } from '../../api/types'
import { MatchLineupForm } from './MatchLineupForm'
import { PlayerBench } from './PlayerBench'
import { SoccerPitch } from './SoccerPitch'
import { useLineupSelections } from './useLineupSelections'
import { UnavailableStartersWarning } from './UnavailableStartersWarning'
import { useSquadUnavailability } from './useSquadUnavailability'

// A row being edited in the substitution planner - fields start empty ('') until the manager
// picks them, unlike LineupSubstitution which is only ever complete.
interface SubstitutionDraft {
  playerOutId: number | ''
  playerInId: number | ''
  minute: number | ''
}

function initialSubstitutionDrafts(lineup: Lineup): SubstitutionDraft[] {
  return lineup.substitutions.map((s) => ({ playerOutId: s.playerOutId, playerInId: s.playerInId, minute: s.minute }))
}

export function LineupEditorPage() {
  const { t } = useTranslation()
  const [tab, setTab] = useState(0)
  const { data: loggedUser } = useMe()
  const { data: userTeam, isLoading: teamLoading } = useUserTeam(!!loggedUser)
  const { data: lineup, isLoading: lineupLoading } = useLineup(!!loggedUser && tab === 0)
  const { data: friendlyLineup, isLoading: friendlyLineupLoading } = useFriendlyDefaultLineup(!!loggedUser && tab === 1)

  if (teamLoading || !userTeam) {
    return <Skeleton variant="rectangular" height={300} />
  }

  return (
    <Box>
      <Tabs value={tab} onChange={(_, next: number) => setTab(next)} sx={{ mb: { xs: 1, sm: 2 } }}>
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
      <Typography variant="body2" color="text.secondary" sx={{ mb: { xs: 1, sm: 2 } }}>
        {t('lineup.friendlyHint')}
      </Typography>
      <MatchLineupForm
        titleKey="lineup.title"
        titleValues={{ teamName: team.name }}
        players={players}
        lineup={lineup}
        onSave={(formation: number, positions: LineupPosition[]) => setFriendlyLineup.mutate({ formation, positions })}
        isSaving={setFriendlyLineup.isPending}
        isError={setFriendlyLineup.isError}
        isSuccess={setFriendlyLineup.isSuccess}
      />
    </Box>
  )
}

function LineupForm({ team, players, lineup }: { team: Team; players: Player[]; lineup: Lineup }) {
  const { t } = useTranslation()
  const unavailability = useSquadUnavailability()
  const {
    formation,
    setFormation,
    counts,
    visibleSelections,
    usedPlayerIds,
    isComplete,
    unavailableStarters,
    benchPlayers,
    getPlayer,
    selectedPlayerId,
    handleSlotClick,
    handleBenchPlayerClick,
    handleBenchAreaClick,
  } = useLineupSelections(players, lineup, unavailability)
  const [substitutionDrafts, setSubstitutionDrafts] = useState<SubstitutionDraft[]>(() => initialSubstitutionDrafts(lineup))
  const setLineup = useSetLineup()

  const starters = players.filter((p) => usedPlayerIds.has(p.id))

  const usedSubOutIds = new Set(substitutionDrafts.map((s) => s.playerOutId).filter((v) => v !== ''))
  const usedSubInIds = new Set(substitutionDrafts.map((s) => s.playerInId).filter((v) => v !== ''))

  const handleAddSubstitution = () => {
    if (substitutionDrafts.length >= MAX_SCHEDULED_SUBSTITUTIONS) return
    setSubstitutionDrafts((prev) => [...prev, { playerOutId: '', playerInId: '', minute: '' }])
  }

  const handleRemoveSubstitution = (index: number) => {
    setSubstitutionDrafts((prev) => prev.filter((_, i) => i !== index))
  }

  const handleSubstitutionChange = (index: number, field: keyof SubstitutionDraft, value: number | '') => {
    setSubstitutionDrafts((prev) => prev.map((s, i) => (i === index ? { ...s, [field]: value } : s)))
  }

  // A draft becomes an actual scheduled substitution only once every field is filled and both
  // players still exist in the current starters/bench (a lineup edit made after picking a
  // substitution can otherwise strand a stale player id) - anything else is silently dropped
  // rather than blocking the whole lineup save over one incomplete row.
  const validSubstitutions: LineupSubstitution[] = substitutionDrafts
    .filter(
      (s): s is { playerOutId: number; playerInId: number; minute: number } =>
        s.playerOutId !== '' &&
        s.playerInId !== '' &&
        s.minute !== '' &&
        usedPlayerIds.has(s.playerOutId) &&
        !usedPlayerIds.has(s.playerInId) &&
        // A suspended/injured player can't come on either (see handlers.validateSubstitutions).
        !unavailability.has(s.playerInId),
    )
    .map((s) => ({ playerOutId: s.playerOutId, playerInId: s.playerInId, minute: s.minute }))

  const handleFormationChange = (event: SelectChangeEvent<number>) => {
    setFormation(Number(event.target.value))
  }

  const handleSubmit = () => {
    if (!isComplete) {
      return
    }
    const positions = visibleSelections.map((s) => ({ position: s.position, playerId: s.playerId as number }))
    setLineup.mutate({ formation, positions, substitutions: validSubstitutions })
  }

  return (
    <Box>
      <Typography variant="h5" sx={{ mb: { xs: 1, sm: 2 } }}>
        {t('lineup.title', { teamName: team.name })}
      </Typography>

      <FormControl size="small" sx={{ mb: { xs: 1, sm: 2 }, minWidth: { xs: '100%', sm: 220 } }}>
        <InputLabel id="formation-label">{t('lineup.formation')}</InputLabel>
        <Select labelId="formation-label" label={t('lineup.formation')} value={formation} onChange={handleFormationChange}>
          {Object.keys(FORMATION_LABEL_KEYS).map((value) => (
            <MenuItem key={value} value={Number(value)}>
              {t(FORMATION_LABEL_KEYS[Number(value)])}
            </MenuItem>
          ))}
        </Select>
      </FormControl>

      <Typography variant="body2" color="text.secondary" sx={{ mb: { xs: 1, sm: 2 } }}>
        {t('lineup.tapHint')}
      </Typography>

      {unavailableStarters.length > 0 && <UnavailableStartersWarning players={unavailableStarters} />}
      {!isComplete && unavailableStarters.length === 0 && (
        <Alert severity="info" sx={{ mb: { xs: 1, sm: 2 } }}>
          {t('lineup.incompleteWarning')}
        </Alert>
      )}
      {setLineup.isError && (
        <Alert severity="error" sx={{ mb: { xs: 1, sm: 2 } }}>
          {t('lineup.saveError')}
        </Alert>
      )}
      {setLineup.isSuccess && (
        <Alert severity="success" sx={{ mb: { xs: 1, sm: 2 } }}>
          {t('lineup.saveSuccess')}
        </Alert>
      )}

      <Grid container spacing={{ xs: 1, sm: 2 }}>
        <Grid size={{ xs: 12, md: 8 }}>
          <SoccerPitch counts={counts} getPlayer={getPlayer} selectedPlayerId={selectedPlayerId} onSlotClick={handleSlotClick} unavailability={unavailability} />
        </Grid>
        <Grid size={{ xs: 12, md: 4 }}>
          <PlayerBench
            players={benchPlayers}
            selectedPlayerId={selectedPlayerId}
            onSelectPlayer={handleBenchPlayerClick}
            onBenchAreaClick={handleBenchAreaClick}
            unavailability={unavailability}
          />
        </Grid>
      </Grid>

      <Paper variant="outlined" sx={{ p: { xs: 1.5, sm: 2 }, mt: { xs: 2, sm: 3 } }}>
        <Typography variant="subtitle1" gutterBottom>
          {t('lineup.substitutions.title')}
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          {t('lineup.substitutions.description', { max: MAX_SCHEDULED_SUBSTITUTIONS })}
        </Typography>
        <Stack spacing={2}>
          {substitutionDrafts.map((draft, index) => (
            <Stack key={index} direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
              <FormControl size="small" sx={{ minWidth: { xs: '100%', sm: 180 } }}>
                <InputLabel id={`sub-out-label-${index}`}>{t('lineup.substitutions.playerOut')}</InputLabel>
                <Select
                  labelId={`sub-out-label-${index}`}
                  label={t('lineup.substitutions.playerOut')}
                  value={draft.playerOutId}
                  onChange={(e) => handleSubstitutionChange(index, 'playerOutId', e.target.value as number)}
                >
                  {starters
                    .filter((p) => p.id === draft.playerOutId || !usedSubOutIds.has(p.id))
                    .map((p) => (
                      <MenuItem key={p.id} value={p.id}>
                        {p.firstName} {p.lastName}
                      </MenuItem>
                    ))}
                </Select>
              </FormControl>
              <FormControl size="small" sx={{ minWidth: { xs: '100%', sm: 180 } }}>
                <InputLabel id={`sub-in-label-${index}`}>{t('lineup.substitutions.playerIn')}</InputLabel>
                <Select
                  labelId={`sub-in-label-${index}`}
                  label={t('lineup.substitutions.playerIn')}
                  value={draft.playerInId}
                  onChange={(e) => handleSubstitutionChange(index, 'playerInId', e.target.value as number)}
                >
                  {benchPlayers
                    .filter((p) => !unavailability.has(p.id))
                    .filter((p) => p.id === draft.playerInId || !usedSubInIds.has(p.id))
                    .map((p) => (
                      <MenuItem key={p.id} value={p.id}>
                        {p.firstName} {p.lastName}
                      </MenuItem>
                    ))}
                </Select>
              </FormControl>
              <TextField
                size="small"
                type="number"
                label={t('lineup.substitutions.minute')}
                value={draft.minute}
                onChange={(e) => {
                  const value = e.target.value === '' ? '' : Number(e.target.value)
                  handleSubstitutionChange(index, 'minute', value)
                }}
                slotProps={{ htmlInput: { min: 1, max: 90 } }}
                sx={{ width: 100 }}
              />
              <Button size="small" color="secondary" onClick={() => handleRemoveSubstitution(index)}>
                {t('lineup.substitutions.remove')}
              </Button>
            </Stack>
          ))}
        </Stack>
        <Button
          size="small"
          variant="outlined"
          sx={{ mt: substitutionDrafts.length > 0 ? 2 : 0 }}
          disabled={substitutionDrafts.length >= MAX_SCHEDULED_SUBSTITUTIONS}
          onClick={handleAddSubstitution}
        >
          {t('lineup.substitutions.add')}
        </Button>
      </Paper>

      <Button
        variant="contained"
        sx={{ mt: { xs: 2, sm: 3 } }}
        disabled={!isComplete || setLineup.isPending}
        onClick={handleSubmit}
      >
        {t('lineup.save')}
      </Button>
    </Box>
  )
}
