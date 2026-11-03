import { useEffect, useMemo, useRef, useState } from 'react'
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
import ToggleButton from '@mui/material/ToggleButton'
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup'
import Typography from '@mui/material/Typography'
import ContentCopyIcon from '@mui/icons-material/ContentCopy'
import { useTranslation } from 'react-i18next'
import {
  FORMATION_LABEL_KEYS,
  MENTALITY_ATTACKING,
  MENTALITY_BALANCED,
  MENTALITY_DEFENSIVE,
  MENTALITY_KEYS,
  POSITION_GOALKEEPER,
  type Lineup,
  type LineupOrder,
  type LineupPosition,
  type LineupRoles,
  type LineupSubstitution,
  type Player,
} from '../../api/types'
import { bestCorePosition } from '../../utils/playerRating'
import { LineupSaveBar } from './LineupSaveBar'
import { LoadPreviousLineupControl } from './LoadPreviousLineupControl'
import { OrdersEditor } from './OrdersEditor'
import { PlayerBench } from './PlayerBench'
import { LINEUP_ROLES, SelectedPlayerPanel, type LineupRole } from './SelectedPlayerPanel'
import { TOUCH } from '../../touchMedia'
import { SoccerPitch } from './SoccerPitch'
import { SubstitutionsEditor } from './SubstitutionsEditor'
import { useLineupSelections } from './useLineupSelections'
import { useOrderDrafts } from './useOrderDrafts'
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
  onSave: (
    formation: number,
    positions: LineupPosition[],
    substitutions: LineupSubstitution[],
    mentality: number,
    orders: LineupOrder[],
    roles: LineupRoles,
  ) => void
  isSaving: boolean
  isError: boolean
  isSuccess: boolean
  // Shows the planned-substitutions section below the pitch/bench editor. Set by every caller:
  // the per-match forms (MatchDetailsPage, CupTieDetailsPage, FriendlyMatchPage), which schedule up
  // to MAX_SCHEDULED_SUBSTITUTIONS for that specific match, and both standing-default editors
  // (LineupEditorPage's league/cup and friendly tabs, each with its own independent set). A match
  // that schedules none falls back to the substitutions of the lineup it plays (see backend
  // common.GetSubstitutionsForMatch/GetSubstitutionsForCupTie and common.GetLineupsForFriendly: a
  // friendly falls back to the friendly default's own set, never the league/cup one).
  substitutionsEnabled?: boolean
  // Shows the conditional-orders section (see OrdersEditor) - every lineup that can schedule
  // substitutions except the friendly default, which carries no orders.
  ordersEnabled?: boolean
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
  ordersEnabled,
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
  const { orderDrafts, handleAddOrder, handleRemoveOrder, handleOrderChange, resetOrderDrafts, ordersValid, validOrders } = useOrderDrafts(
    lineup,
    usedPlayerIds,
  )

  const starters = players.filter((p) => usedPlayerIds.has(p.id))
  // How the side plays (openspec/specs/match-simulation "Team mentality"), saved with the lineup.
  const [mentality, setMentality] = useState(lineup.mentality ?? MENTALITY_BALANCED)
  // The optional captain and set-piece takers (openspec match-simulation "Lineup roles"), saved
  // with the lineup; a role whose player has left the squad reads automatic.
  const [roles, setRoles] = useState<LineupRoles>(() => squadRoles(lineup, players))

  // Who would take each unavailable starter's slot if the match kicked off now (computed by the
  // server for the saved lineup). Dropped once the manager has fielded the replacement himself.
  const replacements = useMemo(() => {
    const byStarter = new Map<number, Player>()
    for (const r of lineup.projectedReplacements ?? []) {
      const replacement = players.find((p) => p.id === r.playerInId)
      if (replacement && !usedPlayerIds.has(replacement.id)) {
        byStarter.set(r.playerOutId, replacement)
      }
    }
    return byStarter
  }, [lineup.projectedReplacements, players, usedPlayerIds])

  // See LineupEditorPage's identical handler: loading a different match's lineup can un-start a
  // player a planned substitution still names as playerOutId, so any in-progress drafts are
  // cleared alongside it rather than left stranded.
  const handleLoadPreviousLineup = (loaded: Lineup) => {
    loadLineup(loaded)
    resetSubstitutionDrafts()
    resetOrderDrafts()
    setMentality(loaded.mentality ?? MENTALITY_BALANCED)
    setRoles(squadRoles(loaded, players))
  }

  const handleFormationChange = (event: SelectChangeEvent<number>) => {
    setFormation(Number(event.target.value))
  }

  // A goalkeeper never takes set pieces: a taker who ends up in goal is dropped back to automatic
  // rather than blocking the save.
  const goalkeeperId = visibleSelections.find((s) => s.position === POSITION_GOALKEEPER)?.playerId ?? null
  const isGoalkeeper = (player: Player) =>
    player.id === goalkeeperId || (!usedPlayerIds.has(player.id) && bestCorePosition(player) === POSITION_GOALKEEPER)
  const savableRoles: LineupRoles = {
    captainId: roles.captainId,
    penaltyTakerId: takerOrAutomatic(roles.penaltyTakerId, players, isGoalkeeper),
    freeKickTakerId: takerOrAutomatic(roles.freeKickTakerId, players, isGoalkeeper),
  }
  const handleToggleRole = (role: LineupRole, playerId: number) =>
    setRoles((prev) => ({ ...prev, [role]: prev[role] === playerId ? undefined : playerId }))
  const roleBadges = new Map<number, string>()
  for (const { role, key } of LINEUP_ROLES) {
    const id = savableRoles[role]
    if (id !== undefined) {
      const label = t(`lineup.roles.badge.${key}`)
      roleBadges.set(id, roleBadges.has(id) ? `${roleBadges.get(id)} ${label}` : label)
    }
  }

  const canSave = isComplete && (!ordersEnabled || ordersValid)
  const handleSubmit = () => {
    if (!canSave) return
    const positions = visibleSelections.map((s) => ({ position: s.position, playerId: s.playerId as number }))
    onSave(formation, positions, substitutionsEnabled ? validSubstitutions : [], mentality, ordersEnabled ? validOrders : [], savableRoles)
  }

  // "Unsaved changes" = the editor differs from what was last saved (or loaded, before the first
  // save). The snapshot is taken when Save is pressed and promoted to the baseline once the save
  // reports success, so an edit made while the request is in flight still counts as unsaved.
  const currentKey = JSON.stringify({
    formation,
    mentality,
    roles: savableRoles,
    visibleSelections,
    drafts: substitutionsEnabled ? substitutionDrafts : [],
    orders: ordersEnabled ? orderDrafts : [],
  })
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
      <Stack direction="row" spacing={1} sx={{ mb: 1, alignItems: 'center' }}>
        <Typography variant="caption" color="text.secondary" component="div" id="match-mentality-label" sx={{ flexShrink: 0 }}>
          {t('lineup.mentality.label')}
        </Typography>
        <ToggleButtonGroup
          value={mentality}
          exclusive
          size="small"
          fullWidth
          aria-labelledby="match-mentality-label"
          onChange={(_, value: number | null) => {
            if (value !== null) setMentality(value)
          }}
          sx={{ maxWidth: { sm: 360 }, '& .MuiToggleButton-root': { py: 0.25, px: 0.75, minHeight: 32, [TOUCH]: { minHeight: 36 }, textTransform: 'none', lineHeight: 1.2 } }}
        >
          {[MENTALITY_DEFENSIVE, MENTALITY_BALANCED, MENTALITY_ATTACKING].map((value) => (
            <ToggleButton key={value} value={value}>
              {t(`lineup.mentality.${MENTALITY_KEYS[value]}`)}
            </ToggleButton>
          ))}
        </ToggleButtonGroup>
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
            replacements={replacements}
            roleBadges={roleBadges}
          />
          {selectedPlayer && (
            <SelectedPlayerPanel
              player={selectedPlayer}
              onClear={clearSelection}
              roles={savableRoles}
              onToggleRole={(role) => handleToggleRole(role, selectedPlayer.id)}
              isGoalkeeper={isGoalkeeper(selectedPlayer)}
            />
          )}
          <RolesSummary roles={savableRoles} players={players} />
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

      {ordersEnabled && (
        <OrdersEditor
          starters={starters}
          benchPlayers={benchPlayers}
          unavailability={unavailability}
          drafts={orderDrafts}
          onAdd={handleAddOrder}
          onRemove={handleRemoveOrder}
          onChange={handleOrderChange}
        />
      )}

      <LineupSaveBar
        dirty={dirty}
        canSave={canSave}
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

/** The lineup's roles, dropping any whose player is no longer in the squad. */
function squadRoles(lineup: Lineup, players: Player[]): LineupRoles {
  const inSquad = (id?: number) => (id !== undefined && players.some((p) => p.id === id) ? id : undefined)
  return { captainId: inSquad(lineup.captainId), penaltyTakerId: inSquad(lineup.penaltyTakerId), freeKickTakerId: inSquad(lineup.freeKickTakerId) }
}

function takerOrAutomatic(id: number | undefined, players: Player[], isGoalkeeper: (p: Player) => boolean): number | undefined {
  const player = id === undefined ? undefined : players.find((p) => p.id === id)
  return player && !isGoalkeeper(player) ? id : undefined
}

/** "Captain: Pérez · Penalties: automatic · Free kicks: Ruiz", under the pitch. */
function RolesSummary({ roles, players }: { roles: LineupRoles; players: Player[] }) {
  const { t } = useTranslation()
  const name = (id?: number) => {
    const player = id === undefined ? undefined : players.find((p) => p.id === id)
    return player ? player.lastName : t('lineup.roles.automatic')
  }
  return (
    <Typography variant="caption" color="text.secondary" component="p" sx={{ mt: 1, mb: 0 }} data-testid="lineup-roles-summary">
      {LINEUP_ROLES.map(({ role, key }) => `${t(`lineup.roles.${key}`)}: ${name(roles[role])}`).join(' · ')}
    </Typography>
  )
}
