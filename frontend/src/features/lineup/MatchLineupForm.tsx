import Alert from '@mui/material/Alert'
import Button from '@mui/material/Button'
import FormControl from '@mui/material/FormControl'
import Grid from '@mui/material/Grid'
import InputLabel from '@mui/material/InputLabel'
import MenuItem from '@mui/material/MenuItem'
import Paper from '@mui/material/Paper'
import Select, { type SelectChangeEvent } from '@mui/material/Select'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { FORMATION_LABEL_KEYS, type Lineup, type LineupPosition, type LineupSubstitution, type Player } from '../../api/types'
import { PlayerBench } from './PlayerBench'
import { SoccerPitch } from './SoccerPitch'
import { SubstitutionsEditor } from './SubstitutionsEditor'
import { useLineupSelections } from './useLineupSelections'
import { useSubstitutionDrafts } from './useSubstitutionDrafts'
import { UnavailableStartersWarning } from './UnavailableStartersWarning'
import { useSquadUnavailability } from './useSquadUnavailability'

export interface MatchLineupFormProps {
  // i18n key rendered as the form's title, e.g. "friendlies.lineup.title" - interpolated with
  // titleValues (typically just { teamName }).
  titleKey: string
  titleValues?: Record<string, unknown>
  players: Player[]
  lineup: Lineup
  onSave: (formation: number, positions: LineupPosition[], substitutions: LineupSubstitution[]) => void
  isSaving: boolean
  isError: boolean
  isSuccess: boolean
  // Shows the planned-substitutions section below the pitch/bench editor, letting the manager
  // schedule up to MAX_SCHEDULED_SUBSTITUTIONS for this specific match rather than only the
  // team's standing default lineup - currently only friendlies support this (see
  // FriendlyMatchPage/common.SetFriendlyLineup); league matches and cup ties still draw their
  // substitutions from the team-wide default (see backend common.SetLineupForMatch/
  // SetCupTieLineup), so they leave this off and onSave's substitutions argument empty.
  substitutionsEnabled?: boolean
}

// A formation/pitch/bench editor scoped to one specific match (league, cup tie, or friendly)
// rather than the team's standing default lineup - shared by MatchDetailsPage (league),
// CupTieDetailsPage (cup), and FriendlyMatchPage (friendly), which each save via their own
// per-kind endpoint (see useSetMatchLineup/useSetCupTieLineup/useSetFriendlyLineup).
export function MatchLineupForm({
  titleKey,
  titleValues,
  players,
  lineup,
  onSave,
  isSaving,
  isError,
  isSuccess,
  substitutionsEnabled,
}: MatchLineupFormProps) {
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
  const {
    substitutionDrafts,
    usedSubOutIds,
    usedSubInIds,
    handleAddSubstitution,
    handleRemoveSubstitution,
    handleSubstitutionChange,
    validSubstitutions,
  } = useSubstitutionDrafts(lineup, usedPlayerIds, unavailability)

  const starters = players.filter((p) => usedPlayerIds.has(p.id))

  const handleFormationChange = (event: SelectChangeEvent<number>) => {
    setFormation(Number(event.target.value))
  }

  const handleSubmit = () => {
    if (!isComplete) return
    const positions = visibleSelections.map((s) => ({ position: s.position, playerId: s.playerId as number }))
    onSave(formation, positions, substitutionsEnabled ? validSubstitutions : [])
  }

  return (
    <Paper variant="outlined" sx={{ p: { xs: 1.5, sm: 2 } }}>
      <Typography variant="h6" sx={{ mb: { xs: 1, sm: 2 } }}>
        {t(titleKey, titleValues)}
      </Typography>

      <FormControl size="small" sx={{ mb: { xs: 1, sm: 2 }, minWidth: { xs: '100%', sm: 220 } }}>
        <InputLabel id="match-formation-label">{t('lineup.formation')}</InputLabel>
        <Select labelId="match-formation-label" label={t('lineup.formation')} value={formation} onChange={handleFormationChange}>
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
      {isError && (
        <Alert severity="error" sx={{ mb: { xs: 1, sm: 2 } }}>
          {t('lineup.saveError')}
        </Alert>
      )}
      {isSuccess && (
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

      {substitutionsEnabled && (
        <SubstitutionsEditor
          starters={starters}
          benchPlayers={benchPlayers}
          unavailability={unavailability}
          drafts={substitutionDrafts}
          usedSubOutIds={usedSubOutIds}
          usedSubInIds={usedSubInIds}
          onAdd={handleAddSubstitution}
          onRemove={handleRemoveSubstitution}
          onChange={handleSubstitutionChange}
        />
      )}

      <Button variant="contained" sx={{ mt: { xs: 2, sm: 3 } }} disabled={!isComplete || isSaving} onClick={handleSubmit}>
        {t('lineup.save')}
      </Button>
    </Paper>
  )
}
