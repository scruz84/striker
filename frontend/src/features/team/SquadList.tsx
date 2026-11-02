import { useMemo, useState } from 'react'
import ArrowDownwardIcon from '@mui/icons-material/ArrowDownward'
import ArrowUpwardIcon from '@mui/icons-material/ArrowUpward'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Chip from '@mui/material/Chip'
import ButtonBase from '@mui/material/ButtonBase'
import Divider from '@mui/material/Divider'
import Drawer from '@mui/material/Drawer'
import FormControl from '@mui/material/FormControl'
import IconButton from '@mui/material/IconButton'
import InputLabel from '@mui/material/InputLabel'
import MenuItem from '@mui/material/MenuItem'
import Paper from '@mui/material/Paper'
import Select, { type SelectChangeEvent } from '@mui/material/Select'
import Stack from '@mui/material/Stack'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import CloseIcon from '@mui/icons-material/Close'
import { Link as RouterLink, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useReleasePlayer } from '../../api/hooks'
import {
  COUNTRY_NAME_KEYS,
  MAX_CONTRACT_YEARS_REMAINING,
  POSITION_DEFENSE,
  POSITION_GOALKEEPER,
  POSITION_MIDFIELD,
  POSITION_SHORT_LABEL_KEYS,
  POSITION_STRIKER,
  type LoanedInPlayer,
  type Player,
  type PlayerUnavailability,
} from '../../api/types'
import { CartoonAvatar } from '../../components/avatar/CartoonAvatar'
import { ContractRenewalDialog } from '../../components/ContractRenewalDialog'
import { useConfirm } from '../../components/confirmDialogContext'
import { ListPlayerForTransferDialog } from '../../components/ListPlayerForTransferDialog'
import { SkillChipsRow } from '../../components/SkillChipsRow'
import { TransferOfferDialog } from '../../components/TransferOfferDialog'
import { CountryFlag } from '../../components/CountryFlag'
import { formatCurrency } from '../../utils/formatCurrency'
import { bestCorePosition, overallRating, ratingTierColor } from '../../utils/playerRating'
import { TransferListedBadge } from '../lineup/TransferListedBadge'
import { UnavailabilityBadge } from '../lineup/UnavailabilityBadge'

type SortDirection = 'asc' | 'desc'
type SquadSortField = 'name' | 'age' | 'wage' | 'skill'

function compareSquadPlayers(a: Player, b: Player, field: SquadSortField): number {
  switch (field) {
    case 'age':
      return a.age - b.age
    case 'wage':
      return a.wage - b.wage
    case 'skill':
      return (a.averageSkill ?? overallRating(a)) - (b.averageSkill ?? overallRating(b))
    default:
      return `${a.firstName} ${a.lastName}`.localeCompare(`${b.firstName} ${b.lastName}`)
  }
}

const POSITION_GROUPS = [
  { position: POSITION_GOALKEEPER, labelKey: 'team.squadGroups.goalkeepers' },
  { position: POSITION_DEFENSE, labelKey: 'team.squadGroups.defenders' },
  { position: POSITION_MIDFIELD, labelKey: 'team.squadGroups.midfielders' },
  { position: POSITION_STRIKER, labelKey: 'team.squadGroups.strikers' },
] as const

interface SquadListProps {
  players: Player[]
  isOwnTeam: boolean
  unavailability: Map<number, PlayerUnavailability>
  transferListed: Set<number>
  /** Players of `players` who are only on loan to this club - they get a "Loan" chip. */
  loanedIn?: LoanedInPlayer[]
}

/** The squad as compact one-row cards (~64px each), grouped by position with a counter for the
 * manager's own team. Everything beyond name, age, position and rating - skills, contract,
 * actions - lives in a bottom sheet opened by tapping the card, so the whole squad fits in a couple
 * of screens and every figure there is labelled. */
export function SquadList({ players, isOwnTeam, unavailability, transferListed, loanedIn = [] }: SquadListProps) {
  const { t } = useTranslation()
  const loanedInById = useMemo(() => new Map(loanedIn.map((loan) => [loan.playerId, loan])), [loanedIn])
  const [sortField, setSortField] = useState<SquadSortField>('name')
  const [sortDir, setSortDir] = useState<SortDirection>('asc')
  const [openPlayerId, setOpenPlayerId] = useState<number | null>(null)

  const sorted = useMemo(
    () => [...players].sort((a, b) => (sortDir === 'asc' ? 1 : -1) * compareSquadPlayers(a, b, sortField)),
    [players, sortField, sortDir],
  )
  const openPlayer = players.find((p) => p.id === openPlayerId)

  // Grouping needs the real skills - a rival's rostered players only expose an aggregate (every
  // individual skill is zeroed), so for them the list stays flat.
  const groups = isOwnTeam
    ? POSITION_GROUPS.map((group) => ({
        ...group,
        players: sorted.filter((p) => bestCorePosition(p) === group.position),
      })).filter((group) => group.players.length > 0)
    : [{ position: -1, labelKey: '', players: sorted }]

  const renderCard = (player: Player) => (
    <SquadPlayerCard
      key={player.id}
      player={player}
      isOwnTeam={isOwnTeam}
      unavailability={isOwnTeam ? unavailability.get(player.id) : undefined}
      forSale={isOwnTeam && transferListed.has(player.id)}
      loanedIn={loanedInById.get(player.id)}
      onOpen={() => setOpenPlayerId(player.id)}
    />
  )

  return (
    <Box>
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mb: 1.5 }}>
        <FormControl sx={{ minWidth: 0, flex: 1, maxWidth: 240 }} size="small">
          <InputLabel id="squad-sort-label">{t('team.sortBy')}</InputLabel>
          <Select
            labelId="squad-sort-label"
            label={t('team.sortBy')}
            value={sortField}
            onChange={(event: SelectChangeEvent) => setSortField(event.target.value as SquadSortField)}
          >
            <MenuItem value="name">{t('team.player')}</MenuItem>
            <MenuItem value="age">{t('team.age')}</MenuItem>
            <MenuItem value="wage">{t('team.wage')}</MenuItem>
            <MenuItem value="skill">{t('team.averageSkill')}</MenuItem>
          </Select>
        </FormControl>
        <IconButton onClick={() => setSortDir((prev) => (prev === 'asc' ? 'desc' : 'asc'))} aria-label={t(sortDir === 'asc' ? 'lineup.sortAscending' : 'lineup.sortDescending')}>
          {sortDir === 'asc' ? <ArrowUpwardIcon fontSize="small" /> : <ArrowDownwardIcon fontSize="small" />}
        </IconButton>
      </Stack>

      {groups.map((group) => (
        <Box key={group.position} sx={{ mb: 2 }}>
          {group.labelKey && (
            <Typography variant="overline" color="text.secondary" component="h3" sx={{ display: 'block', mb: 0.5 }}>
              {t(group.labelKey)} ({group.players.length})
            </Typography>
          )}
          <Stack spacing={0.75}>{group.players.map(renderCard)}</Stack>
        </Box>
      ))}

      {openPlayer && (
        <PlayerSheet
          player={openPlayer}
          isOwnTeam={isOwnTeam}
          unavailability={isOwnTeam ? unavailability.get(openPlayer.id) : undefined}
          forSale={isOwnTeam && transferListed.has(openPlayer.id)}
          onClose={() => setOpenPlayerId(null)}
        />
      )}
    </Box>
  )
}

function SquadPlayerCard({
  player,
  isOwnTeam,
  unavailability,
  forSale,
  loanedIn,
  onOpen,
}: {
  player: Player
  isOwnTeam: boolean
  unavailability?: PlayerUnavailability
  forSale?: boolean
  loanedIn?: LoanedInPlayer
  onOpen: () => void
}) {
  const { t } = useTranslation()
  const theme = useTheme()
  const rating = player.averageSkill ?? overallRating(player)

  return (
    <Paper variant="outlined">
      <ButtonBase onClick={onOpen} sx={{ width: '100%', display: 'flex', gap: 1.5, p: 1, minHeight: 64, textAlign: 'left', justifyContent: 'flex-start' }} aria-label={`${player.firstName} ${player.lastName}`}>
        <CartoonAvatar seed={player.id} birthCountry={player.birthCountry} size={44} />
        <Box sx={{ minWidth: 0, flex: 1 }}>
          <Typography variant="body2" sx={{ fontWeight: 600, lineHeight: 1.25, overflowWrap: 'anywhere' }}>
            {player.firstName} {player.lastName}
          </Typography>
          <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center', flexWrap: 'wrap', rowGap: 0.25 }}>
            {isOwnTeam && (
              <Typography variant="caption" color="text.secondary" translate="no" className="notranslate">
                {t(POSITION_SHORT_LABEL_KEYS[bestCorePosition(player)])} ·
              </Typography>
            )}
            <Typography variant="caption" color="text.secondary">
              {t('lineup.ageShort', { age: player.age })}
            </Typography>
            <CountryFlag code={player.birthCountry} height={14} />
            {unavailability && <UnavailabilityBadge unavailability={unavailability} size={18} />}
            {forSale && <TransferListedBadge size={18} />}
            {loanedIn && (
              <Tooltip title={t('team.loanedInTooltip', { team: loanedIn.parentTeamName, season: loanedIn.returnSeason })}>
                <Chip size="small" variant="outlined" color="info" label={t('team.loanedInChip')} sx={{ height: 18, fontSize: '0.6875rem' }} />
              </Tooltip>
            )}
          </Stack>
        </Box>
        <Stack sx={{ alignItems: 'center', flexShrink: 0, pr: 0.5 }}>
          <Typography variant="stat" sx={{ fontSize: '1.375rem', color: ratingTierColor(rating, theme.palette) }}>
            {rating}
          </Typography>
          <Typography variant="chipLabel" color="text.secondary">
            {t('team.averageSkill')}
          </Typography>
        </Stack>
      </ButtonBase>
    </Paper>
  )
}

/** Bottom sheet with everything about one player that does not fit in the list row: skills (own
 * squad), contract with labelled figures, and the actions - with the destructive one set apart. */
function PlayerSheet({
  player,
  isOwnTeam,
  unavailability,
  forSale,
  onClose,
}: {
  player: Player
  isOwnTeam: boolean
  unavailability?: PlayerUnavailability
  forSale?: boolean
  onClose: () => void
}) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const confirm = useConfirm()
  const releasePlayer = useReleasePlayer()
  const [renewalOpen, setRenewalOpen] = useState(false)
  const [listOpen, setListOpen] = useState(false)
  const [offerOpen, setOfferOpen] = useState(false)
  const maxContract = player.contractYearsRemaining >= MAX_CONTRACT_YEARS_REMAINING

  const handleRelease = async () => {
    const ok = await confirm({
      message: t('team.confirmRelease', { name: `${player.firstName} ${player.lastName}` }),
      confirmLabel: t('team.releasePlayer'),
      destructive: true,
    })
    if (ok) {
      releasePlayer.mutate(player.id, { onSuccess: onClose })
    }
  }

  return (
    <>
      <Drawer
        anchor="bottom"
        open
        onClose={onClose}
        slotProps={{ paper: { sx: { borderTopLeftRadius: 20, borderTopRightRadius: 20, maxHeight: '85dvh', maxWidth: 560, mx: 'auto', width: '100%' } } }}
      >
        <Box sx={{ p: 2, pb: 'max(16px, env(safe-area-inset-bottom))', overflowY: 'auto' }}>
          <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center', mb: 1.5 }}>
            <CartoonAvatar seed={player.id} birthCountry={player.birthCountry} size={48} />
            <Box sx={{ minWidth: 0, flex: 1 }}>
              <Typography variant="subtitle1" sx={{ fontWeight: 700, overflowWrap: 'anywhere', lineHeight: 1.25 }}>
                {player.firstName} {player.lastName}
              </Typography>
              <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center' }}>
                <Typography variant="body2" color="text.secondary">
                  {t('lineup.ageShort', { age: player.age })}
                </Typography>
                {player.birthCountry && COUNTRY_NAME_KEYS[player.birthCountry] && (
                  <Typography variant="body2" color="text.secondary">
                    · {t(COUNTRY_NAME_KEYS[player.birthCountry])}
                  </Typography>
                )}
                {unavailability && <UnavailabilityBadge unavailability={unavailability} size={18} />}
                {forSale && <TransferListedBadge size={18} />}
              </Stack>
            </Box>
            <IconButton onClick={onClose} aria-label={t('common.close')}>
              <CloseIcon />
            </IconButton>
          </Stack>

          {isOwnTeam && <SkillChipsRow player={player} />}

          {isOwnTeam && (
            <Stack direction="row" spacing={3} sx={{ my: 1.5 }}>
              <Box>
                <Typography variant="caption" color="text.secondary" component="div">
                  {t('team.wage')}
                </Typography>
                <Typography variant="body1" sx={{ fontWeight: 600 }}>
                  {t('team.perMatch', { amount: formatCurrency(player.wage) })}
                </Typography>
              </Box>
              <Box>
                <Typography variant="caption" color="text.secondary" component="div">
                  {t('team.contract')}
                </Typography>
                <Typography variant="body1" sx={{ fontWeight: 600 }}>
                  {t('team.contractSeasons', { count: player.contractYearsRemaining })}
                </Typography>
              </Box>
            </Stack>
          )}

          <Stack spacing={1} sx={{ mt: 1.5 }}>
            <Button component={RouterLink} to={`/player/${player.id}`} variant="outlined" onClick={onClose}>
              {t('playerCard.viewProfile')}
            </Button>
            {isOwnTeam ? (
              <>
                <Button variant="contained" onClick={() => setRenewalOpen(true)} disabled={maxContract}>
                  {t('team.renewContract')}
                </Button>
                {maxContract && (
                  <Typography variant="caption" color="text.secondary">
                    {t('team.renewDisabledReason', { count: MAX_CONTRACT_YEARS_REMAINING })}
                  </Typography>
                )}
                <Stack direction="row" spacing={1}>
                  <Button fullWidth variant="outlined" onClick={() => setListOpen(true)} disabled={forSale}>
                    {t('team.listForSale')}
                  </Button>
                  <Button
                    fullWidth
                    variant="outlined"
                    onClick={() => {
                      onClose()
                      navigate('/transfers?tab=loans')
                    }}
                  >
                    {t('team.loanOut')}
                  </Button>
                </Stack>
                <Divider sx={{ my: 1 }} />
                <Button color="error" variant="text" onClick={handleRelease} disabled={releasePlayer.isPending}>
                  {t('team.releasePlayer')}
                </Button>
                {releasePlayer.isError && <Alert severity="error">{t('team.releaseError')}</Alert>}
              </>
            ) : (
              <Button variant="contained" onClick={() => setOfferOpen(true)}>
                {t('team.makeOffer')}
              </Button>
            )}
          </Stack>
        </Box>
      </Drawer>
      {isOwnTeam && <ContractRenewalDialog player={player} open={renewalOpen} onClose={() => setRenewalOpen(false)} />}
      {isOwnTeam && <ListPlayerForTransferDialog player={player} open={listOpen} onClose={() => setListOpen(false)} />}
      {!isOwnTeam && <TransferOfferDialog player={player} open={offerOpen} onClose={() => setOfferOpen(false)} />}
    </>
  )
}
