import { useEffect, useRef, useState } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Collapse from '@mui/material/Collapse'
import FormControl from '@mui/material/FormControl'
import Grid from '@mui/material/Grid'
import InputLabel from '@mui/material/InputLabel'
import MenuItem from '@mui/material/MenuItem'
import Paper from '@mui/material/Paper'
import Select, { type SelectChangeEvent } from '@mui/material/Select'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import ContentCopyIcon from '@mui/icons-material/ContentCopy'
import { useTranslation } from 'react-i18next'
import { FORMATION_LABEL_KEYS, type Lineup, type LineupPosition, type LineupSubstitution, type Player } from '../../api/types'
import { LineupSaveBar } from './LineupSaveBar'
import { LoadPreviousLineupControl } from './LoadPreviousLineupControl'
import { PlayerBench } from './PlayerBench'
import { SelectedPlayerPanel } from './SelectedPlayerPanel'
import { SoccerPitch } from './SoccerPitch'
import { SubstitutionsEditor } from './SubstitutionsEditor'
import { useLineupSelections } from './useLineupSelections'
import { useSubstitutionDrafts } from './useSubstitutionDrafts'
import { UnavailableStartersWarning } from './UnavailableStartersWarning'
import { useSquadUnavailability } from './useSquadUnavailability'
import { useSquadTransferListed } from './useSquadTransferListed'
import { useUnsavedChangesGuard } from './useUnsavedChangesGuard'

export interface MatchLineupFormProps {
  // i18n key rendered as the form's title, e.g. "friendlies.lineup.title" - interpolated with
  // titleValues (typically just { teamName }).
  titleKey: string
  titleValues?: Record<string, unknown>
  // The team this lineup belongs to - lets the form offer LoadPreviousLineupControl (copy a past
  // league match's lineup straight in). Every per-match caller (MatchDetailsPage,
  // CupTieDetailsPage, FriendlyMatchPage) already has the team loaded to pass here.
  teamId: number
  players: Player[]
  lineup: Lineup
  onSave: (formation: number, positions: LineupPosition[], substitutions: LineupSubstitution[]) => void
  isSaving: boolean
  isError: boolean
  isSuccess: boolean
  // Shows the planned-substitutions section below the pitch/bench editor, letting the manager
  // schedule up to MAX_SCHEDULED_SUBSTITUTIONS for this specific match rather than only the
  // team's standing default lineup - set by every per-match caller (MatchDetailsPage,
  // CupTieDetailsPage, FriendlyMatchPage). A team that doesn't schedule any for this particular
  // match still falls back to its team-wide default (see backend
  // common.GetSubstitutionsForMatch/GetSubstitutionsForCupTie/GetFriendlyLineupSubstitutions).
  // The team's own standing-default editor (LineupEditorPage) renders SubstitutionsEditor
  // directly instead of through this flag.
  substitutionsEnabled?: boolean
  // A red-card suspension carries no sanction into a friendly (see useSquadUnavailability) - set
  // this for FriendlyMatchPage's caller so a suspended-only player isn't wrongly flagged/blocked.
  isFriendly?: boolean
  // Set by LineupEditorPage's friendly-default form: a team's standing default lineup is
  // general-purpose rather than scoped to one matchday's squad, so an unavailable starter is shown
  // as informational only and never blocks saving (mirrors LineupForm's league/cup default -
  // see useLineupSelections' enforceAvailability). Per-match forms (FriendlyMatchPage,
  // MatchDetailsPage, CupTieDetailsPage) leave this off and keep the hard block.
  isDefaultLineup?: boolean
  // Render without the outlined card and heading - for a form that already sits directly on its
  // page (the standing-lineup editor), where a wrapping card only costs the pitch ~30px of width.
  embedded?: boolean
}

// A formation/pitch/bench editor scoped to one specific match (league, cup tie, or friendly)
// rather than the team's standing default lineup - shared by MatchDetailsPage (league),
// CupTieDetailsPage (cup), and FriendlyMatchPage (friendly), which each save via their own
// per-kind endpoint (see useSetMatchLineup/useSetCupTieLineup/useSetFriendlyLineup).
export function MatchLineupForm({
  titleKey,
  titleValues,
  teamId,
  players,
  lineup,
  onSave,
  isSaving,
  isError,
  isSuccess,
  substitutionsEnabled,
  isFriendly,
  isDefaultLineup,
  embedded = false,
}: MatchLineupFormProps) {
  const { t } = useTranslation()
  const unavailability = useSquadUnavailability(isFriendly)
  const transferListed = useSquadTransferListed()
  const {
    formation,
    setFormation,
    loadLineup,
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
    clearSelection,
  } = useLineupSelections(players, lineup, unavailability, !isDefaultLineup)
  const {
    substitutionDrafts,
    usedSubOutIds,
    usedSubInIds,
    handleAddSubstitution,
    handleRemoveSubstitution,
    handleSubstitutionChange,
    resetSubstitutionDrafts,
    validSubstitutions,
  } = useSubstitutionDrafts(lineup, usedPlayerIds, unavailability)

  const starters = players.filter((p) => usedPlayerIds.has(p.id))

  // See LineupEditorPage's identical handler: loading a different match's lineup can un-start a
  // player a planned substitution still names as playerOutId, so any in-progress drafts are
  // cleared alongside it rather than left stranded.
  const handleLoadPreviousLineup = (loaded: Lineup) => {
    loadLineup(loaded)
    resetSubstitutionDrafts()
  }

  const handleFormationChange = (event: SelectChangeEvent<number>) => {
    setFormation(Number(event.target.value))
  }

  const handleSubmit = () => {
    if (!isComplete) return
    const positions = visibleSelections.map((s) => ({ position: s.position, playerId: s.playerId as number }))
    onSave(formation, positions, substitutionsEnabled ? validSubstitutions : [])
  }

  // "Unsaved changes" = the editor differs from what was last saved (or loaded, before the first
  // save). The snapshot is taken when Save is pressed and promoted to the baseline once the save
  // reports success, so an edit made while the request is in flight still counts as unsaved.
  const currentKey = JSON.stringify({ formation, visibleSelections, drafts: substitutionsEnabled ? substitutionDrafts : [] })
  const [initialKey] = useState(currentKey)
  const pendingKey = useRef<string | null>(null)
  const [savedKey, setSavedKey] = useState<string | null>(null)
  const dirty = currentKey !== (savedKey ?? initialKey)
  useEffect(() => {
    if (isSuccess && pendingKey.current !== null) {
      setSavedKey(pendingKey.current)
      pendingKey.current = null
    }
  }, [isSuccess])
  useUnsavedChangesGuard(dirty)

  const [copyOpen, setCopyOpen] = useState(false)
  const selectedPlayer = selectedPlayerId === null ? undefined : players.find((p) => p.id === selectedPlayerId)

  const content = (
    <>
      {!embedded && (
        <Typography variant="h6" sx={{ mb: { xs: 1, sm: 2 } }}>
          {t(titleKey, titleValues)}
        </Typography>
      )}

      <Stack direction="row" spacing={1} sx={{ mb: 1, alignItems: 'center' }}>
        <FormControl sx={{ flex: 1, minWidth: 0, maxWidth: { sm: 280 } }}>
          <InputLabel id="match-formation-label">{t('lineup.formation')}</InputLabel>
          <Select labelId="match-formation-label" label={t('lineup.formation')} value={formation} onChange={handleFormationChange}>
            {Object.keys(FORMATION_LABEL_KEYS).map((value) => (
              <MenuItem key={value} value={Number(value)}>
                {t(FORMATION_LABEL_KEYS[Number(value)])}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        <Button
          variant={copyOpen ? 'contained' : 'outlined'}
          startIcon={<ContentCopyIcon />}
          onClick={() => setCopyOpen((open) => !open)}
          aria-expanded={copyOpen}
          sx={{ flexShrink: 0 }}
        >
          {t('lineup.copyFromMatch')}
        </Button>
      </Stack>
      <Collapse in={copyOpen} unmountOnExit>
        <LoadPreviousLineupControl
          teamId={teamId}
          onLoad={(loaded) => {
            handleLoadPreviousLineup(loaded)
            setCopyOpen(false)
          }}
        />
      </Collapse>

      <Typography variant="caption" color="text.secondary" component="p" sx={{ mt: 0, mb: 1.5 }}>
        {t('lineup.tapHint')}
      </Typography>

      {unavailableStarters.length > 0 && (
        <UnavailableStartersWarning players={unavailableStarters} blocking={!isDefaultLineup} />
      )}
      {isError && (
        <Alert severity="error" sx={{ mb: { xs: 1, sm: 2 } }}>
          {t('lineup.saveError')}
        </Alert>
      )}

      <Grid container spacing={{ xs: 1, sm: 2 }}>
        <Grid size={{ xs: 12, md: 8 }}>
          <SoccerPitch
            counts={counts}
            getPlayer={getPlayer}
            selectedPlayerId={selectedPlayerId}
            onSlotClick={handleSlotClick}
            unavailability={unavailability}
            transferListed={transferListed}
          />
          {selectedPlayer && <SelectedPlayerPanel player={selectedPlayer} onClear={clearSelection} />}
        </Grid>
        <Grid size={{ xs: 12, md: 4 }}>
          <PlayerBench
            players={benchPlayers}
            selectedPlayerId={selectedPlayerId}
            onSelectPlayer={handleBenchPlayerClick}
            onBenchAreaClick={handleBenchAreaClick}
            unavailability={unavailability}
            transferListed={transferListed}
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

      <LineupSaveBar
        dirty={dirty}
        canSave={isComplete}
        saving={isSaving}
        saved={isSuccess && !dirty}
        hint={!isComplete && unavailableStarters.length === 0 ? t('lineup.cantSaveHint') : undefined}
        onSave={() => {
          pendingKey.current = currentKey
          handleSubmit()
        }}
      />
    </>
  )

  return embedded ? <Box>{content}</Box> : <Paper variant="outlined" sx={{ p: { xs: 1.5, sm: 2 } }}>{content}</Paper>
}
