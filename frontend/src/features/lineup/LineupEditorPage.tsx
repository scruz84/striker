import { useState } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import FormControl from '@mui/material/FormControl'
import Grid from '@mui/material/Grid'
import InputLabel from '@mui/material/InputLabel'
import MenuItem from '@mui/material/MenuItem'
import Select, { type SelectChangeEvent } from '@mui/material/Select'
import Skeleton from '@mui/material/Skeleton'
import Tab from '@mui/material/Tab'
import Tabs from '@mui/material/Tabs'
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
import { FORMATION_LABEL_KEYS } from '../../api/types'
import type { Lineup, LineupPosition, Player, Team } from '../../api/types'
import { LoadPreviousLineupControl } from './LoadPreviousLineupControl'
import { MatchLineupForm } from './MatchLineupForm'
import { PlayerBench } from './PlayerBench'
import { SoccerPitch } from './SoccerPitch'
import { SubstitutionsEditor } from './SubstitutionsEditor'
import { useLineupSelections } from './useLineupSelections'
import { useSubstitutionDrafts } from './useSubstitutionDrafts'
import { UnavailableStartersWarning } from './UnavailableStartersWarning'
import { useSquadUnavailability } from './useSquadUnavailability'
import { useSquadTransferListed } from './useSquadTransferListed'

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
        isDefaultLineup
      />
    </Box>
  )
}

function LineupForm({ team, players, lineup }: { team: Team; players: Player[]; lineup: Lineup }) {
  const { t } = useTranslation()
  const unavailability = useSquadUnavailability()
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
  } = useLineupSelections(players, lineup, unavailability, false)
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
  const setLineup = useSetLineup()

  // Loading a different match's lineup can un-start a player a planned substitution still names
  // as playerOutId - clearing the drafts alongside it avoids a stale, unfillable row rather than
  // leaving the manager to notice and remove it by hand.
  const handleLoadPreviousLineup = (loaded: Lineup) => {
    loadLineup(loaded)
    resetSubstitutionDrafts()
  }

  const starters = players.filter((p) => usedPlayerIds.has(p.id))

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

      <LoadPreviousLineupControl teamId={team.id} onLoad={handleLoadPreviousLineup} />

      {unavailableStarters.length > 0 && <UnavailableStartersWarning players={unavailableStarters} blocking={false} />}
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
          <SoccerPitch
            counts={counts}
            getPlayer={getPlayer}
            selectedPlayerId={selectedPlayerId}
            onSlotClick={handleSlotClick}
            unavailability={unavailability}
            transferListed={transferListed}
          />
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
