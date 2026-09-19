import { useState, type ReactNode } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Chip from '@mui/material/Chip'
import Link from '@mui/material/Link'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import useMediaQuery from '@mui/material/useMediaQuery'
import { useTheme } from '@mui/material/styles'
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useTransferWindowStatus } from '../api/hooks'
import { COUNTRY_NAME_KEYS, POSITION_LABEL_KEYS, UNAVAILABLE_INJURY, type PlayerDetails } from '../api/types'
import { overallRating, ratingTierColor } from '../utils/playerRating'
import { countryFlag } from '../utils/countryFlag'
import { formatCurrency } from '../utils/formatCurrency'
import { CartoonAvatar } from './avatar/CartoonAvatar'
import { ListPlayerForTransferDialog } from './ListPlayerForTransferDialog'
import { TeamBadge } from './TeamBadge'

const RING_SIZE_MOBILE = 64
const RING_SIZE_DESKTOP = 96
const RING_STROKE = 6

/** The player details page's bio column - portrait with a rating ring, status alerts, and a
 * spec-sheet of team/division/contract/stats. Sits beside PlayerAttributesPanel instead of the
 * page spreading the same facts across a header, a banner list and five separate stat tiles. */
export function PlayerProfileRail({
  player,
  canSeeFinancials = true,
  canSeeSkills = true,
}: {
  player: PlayerDetails
  /** Whether the viewer may see this player's wage and contract length - false when someone else
   * owns this player's team (see PlayerDetailsPage). Defaults true for other callers (e.g. the
   * squad page's own PlayerRow, which already gates the whole component by team ownership). */
  canSeeFinancials?: boolean
  /** Whether the viewer may see this player's skill ratings/traits/best position - false when
   * someone else owns this player's team and hasn't listed them for transfer (see
   * PlayerDetailsPage and the backend's common.Player.RedactSkills, which zeroes the underlying
   * fields for the same case, so this only decides whether to render them). Defaults true for
   * other callers, same reasoning as canSeeFinancials. */
  canSeeSkills?: boolean
}) {
  const { t } = useTranslation()
  const theme = useTheme()
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'))
  const ringSize = isMobile ? RING_SIZE_MOBILE : RING_SIZE_DESKTOP
  const ringRadius = (ringSize - RING_STROKE) / 2
  const ringCircumference = 2 * Math.PI * ringRadius
  const badgeSize = isMobile ? 24 : 32
  const isFreeAgent = player.teamId === 0
  const overall = overallRating(player)
  const ringColor = ratingTierColor(overall, theme.palette)
  const flag = countryFlag(player.birthCountry)

  // Listing straight from this page needs the same eligibility the transfer market's own "List a
  // player" form already enforces server-side (see backend common.CreateTransferListing) - not
  // borrowed on loan, not already listed for transfer or loan, and the window open - checked here
  // too so the button doesn't invite a click that's just going to fail.
  const { data: windowStatus } = useTransferWindowStatus(canSeeFinancials && !isFreeAgent)
  const canListForTransfer =
    canSeeFinancials &&
    !isFreeAgent &&
    !player.onLoanFromTeamId &&
    player.transferListingAskingPrice === undefined &&
    player.loanListingDurationSeasons === undefined &&
    !!windowStatus?.open
  const [listDialogOpen, setListDialogOpen] = useState(false)

  const traits: Array<{ key: string; label: string }> = canSeeSkills
    ? [
        player.header && { key: 'header', label: t('playerCard.traits.aerial') },
        player.quick && { key: 'quick', label: t('playerCard.traits.pace') },
        player.creative && { key: 'creative', label: t('playerCard.traits.creative') },
        player.technical && { key: 'technical', label: t('playerCard.traits.technical') },
      ].filter((trait): trait is { key: string; label: string } => !!trait)
    : []

  return (
    <Paper variant="outlined" sx={{ p: { xs: 1.5, sm: 2.5 } }}>
      <Stack spacing={{ xs: 1, sm: 1.5 }} sx={{ alignItems: 'center', textAlign: 'center' }}>
        <Box sx={{ position: 'relative', width: ringSize, height: ringSize }}>
          {canSeeSkills && (
            <svg
              width={ringSize}
              height={ringSize}
              style={{ position: 'absolute', inset: 0, transform: 'rotate(-90deg)' }}
            >
              <circle
                cx={ringSize / 2}
                cy={ringSize / 2}
                r={ringRadius}
                fill="none"
                stroke={theme.palette.action.hover}
                strokeWidth={RING_STROKE}
              />
              <circle
                cx={ringSize / 2}
                cy={ringSize / 2}
                r={ringRadius}
                fill="none"
                stroke={ringColor}
                strokeWidth={RING_STROKE}
                strokeLinecap="round"
                strokeDasharray={ringCircumference}
                strokeDashoffset={ringCircumference * (1 - overall / 100)}
              />
            </svg>
          )}
          <Box sx={{ position: 'absolute', inset: 0, borderRadius: '50%', overflow: 'hidden' }}>
            <CartoonAvatar seed={player.id} birthCountry={player.birthCountry} size={ringSize} />
          </Box>
          {canSeeSkills && (
            <Box
              sx={{
                position: 'absolute',
                bottom: -4,
                right: -4,
                width: badgeSize,
                height: badgeSize,
                borderRadius: '50%',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                bgcolor: ringColor,
                color: theme.palette.getContrastText(ringColor),
                fontWeight: 700,
                fontSize: isMobile ? '0.7rem' : '0.8rem',
                border: `2px solid ${theme.palette.background.paper}`,
              }}
              title={t('playerCard.overall')}
            >
              {overall}
            </Box>
          )}
        </Box>

        <Box>
          <Typography variant="h6" component="h1" sx={{ fontWeight: 700, lineHeight: 1.2 }}>
            {flag && (
              <Tooltip title={t(COUNTRY_NAME_KEYS[player.birthCountry!] ?? player.birthCountry!)}>
                <Box component="span" sx={{ mr: 0.75 }}>
                  {flag}
                </Box>
              </Tooltip>
            )}
            {player.firstName} {player.lastName}
          </Typography>
          {canSeeSkills && (
            <Typography variant="body2" color="text.secondary">
              {t(POSITION_LABEL_KEYS[player.bestPosition])}
            </Typography>
          )}
        </Box>

        {isFreeAgent && <Chip size="small" label={t('playerPage.freeAgent')} />}

        {traits.length > 0 && (
          <Stack direction="row" spacing={0.5} sx={{ flexWrap: 'wrap', justifyContent: 'center' }}>
            {traits.map((trait) => (
              <Chip key={trait.key} label={trait.label} size="small" variant="outlined" />
            ))}
          </Stack>
        )}

        <PlayerStatusAlerts player={player} />

        <Stack spacing={0.5} sx={{ width: '100%', textAlign: 'left', borderTop: 1, borderColor: 'divider', pt: 1, mt: 0.25 }}>
          {!isFreeAgent && (
            <SpecRow
              label={t('playerPage.profile.team')}
              value={
                <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center', justifyContent: 'flex-end' }}>
                  <TeamBadge teamId={player.teamId} teamName={player.teamName} size={18} />
                  <Link component={RouterLink} to={`/team/${player.teamId}`} underline="hover" variant="body2" sx={{ fontWeight: 600 }}>
                    {player.teamName}
                  </Link>
                </Stack>
              }
            />
          )}
          {!isFreeAgent && (
            <SpecRow
              label={t('playerPage.profile.division')}
              value={
                <Link component={RouterLink} to={`/division/${player.divisionId}`} underline="hover" variant="body2" sx={{ fontWeight: 600 }}>
                  {player.divisionName}
                </Link>
              }
            />
          )}
          <SpecRow label={t('playerPage.profile.age')} value={player.age} />
          {player.birthCountry && (
            <SpecRow label={t('playerPage.profile.birthCountry')} value={t(COUNTRY_NAME_KEYS[player.birthCountry] ?? player.birthCountry)} />
          )}
          {canSeeFinancials && (
            <SpecRow label={t('playerPage.stats.contract')} value={t('team.contractSeasons', { count: player.contractYearsRemaining })} />
          )}
          {canSeeFinancials && <SpecRow label={t('playerPage.stats.wage')} value={formatCurrency(player.wage)} />}
          <SpecRow label={t('playerPage.stats.goals')} value={player.goals} />
          {!isFreeAgent && <SpecRow label={t('playerPage.stats.goalsForTeam')} value={player.goalsForTeam} />}
          <SpecRow label={t('playerPage.stats.yellowCards')} value={player.yellowCards} />
          <SpecRow label={t('playerPage.stats.redCards')} value={player.redCards} />
        </Stack>

        {canListForTransfer && (
          <Button size="small" variant="outlined" fullWidth onClick={() => setListDialogOpen(true)}>
            {t('playerPage.listForTransfer.button')}
          </Button>
        )}
      </Stack>
      <ListPlayerForTransferDialog player={player} open={listDialogOpen} onClose={() => setListDialogOpen(false)} />
    </Paper>
  )
}

function SpecRow({ label, value }: { label: string; value: ReactNode }) {
  return (
    <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', gap: 1 }}>
      <Typography variant="body2" color="text.secondary">
        {label}
      </Typography>
      {typeof value === 'string' || typeof value === 'number' ? (
        <Typography variant="body2" sx={{ fontWeight: 600 }}>
          {value}
        </Typography>
      ) : (
        value
      )}
    </Stack>
  )
}

// One Alert per currently-true status - a player can be e.g. injured AND listed for transfer at
// the same time, so these are independent rather than mutually exclusive.
function PlayerStatusAlerts({ player }: { player: PlayerDetails }) {
  const { t } = useTranslation()
  const alerts: { key: string; severity: 'warning' | 'info'; text: string }[] = []

  if (player.unavailableReason !== undefined) {
    const key = player.unavailableReason === UNAVAILABLE_INJURY ? 'playerPage.status.injured' : 'playerPage.status.suspended'
    alerts.push({ key: 'unavailable', severity: 'warning', text: t(key, { count: player.unavailableMatchesRemaining }) })
  }
  if (player.onLoanFromTeamName) {
    alerts.push({
      key: 'loan',
      severity: 'info',
      text: t('playerPage.status.onLoan', { team: player.onLoanFromTeamName, season: player.onLoanReturnSeason }),
    })
  }
  if (player.transferListingAskingPrice !== undefined) {
    alerts.push({
      key: 'listedTransfer',
      severity: 'info',
      text: t('playerPage.status.listedForTransfer', { price: formatCurrency(player.transferListingAskingPrice) }),
    })
  }
  if (player.loanListingDurationSeasons !== undefined) {
    alerts.push({
      key: 'listedLoan',
      severity: 'info',
      text: t('playerPage.status.listedForLoan', { count: player.loanListingDurationSeasons }),
    })
  }

  if (alerts.length === 0) {
    return null
  }

  return (
    <Stack spacing={1} sx={{ width: '100%' }}>
      {alerts.map((a) => (
        <Alert key={a.key} severity={a.severity} sx={{ py: 0, textAlign: 'left' }}>
          {a.text}
        </Alert>
      ))}
    </Stack>
  )
}
