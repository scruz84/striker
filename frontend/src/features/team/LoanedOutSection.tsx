import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { POSITION_SHORT_LABEL_KEYS, type LoanedOutPlayer } from '../../api/types'
import { CountryFlag } from '../../components/CountryFlag'
import { PlayerNameLink } from '../../components/PlayerNameLink'
import { TeamNameLink } from '../../components/TeamNameLink'
import { formatCurrency } from '../../utils/formatCurrency'
import { bestCorePosition } from '../../utils/playerRating'

interface LoanedOutSectionProps {
  loanedOut: LoanedOutPlayer[]
  /** The lender's own skills are only exposed on his own page - a visitor's view has them zeroed,
   * so the position is only shown to the owner (same rule as the squad list). */
  isOwnTeam: boolean
}

/** The players a club has out on loan, below its squad list: where each one is, when he comes back
 * and the loan's terms. Renders nothing when nobody is out. */
export function LoanedOutSection({ loanedOut, isOwnTeam }: LoanedOutSectionProps) {
  const { t } = useTranslation()
  if (loanedOut.length === 0) return null

  return (
    <Box sx={{ mt: 1 }} data-testid="loaned-out-section">
      <Typography variant="overline" color="text.secondary" component="h3" sx={{ display: 'block', mb: 0.5 }}>
        {t('team.loanedOut.title', { count: loanedOut.length })}
      </Typography>
      <Stack spacing={0.75}>
        {loanedOut.map((loan) => (
          <Paper key={loan.player.id} variant="outlined" sx={{ p: 1.25 }}>
            <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap', rowGap: 0.5 }}>
              <Typography variant="body2" sx={{ fontWeight: 600, overflowWrap: 'anywhere' }}>
                <CountryFlag code={loan.player.birthCountry} height={14} marginRight={0.5} />
                <PlayerNameLink playerId={loan.player.id}>
                  {loan.player.firstName} {loan.player.lastName}
                </PlayerNameLink>
              </Typography>
              {isOwnTeam && (
                <Typography variant="caption" color="text.secondary" translate="no" className="notranslate">
                  {t(POSITION_SHORT_LABEL_KEYS[bestCorePosition(loan.player)])} ·
                </Typography>
              )}
              <Typography variant="caption" color="text.secondary">
                {t('lineup.ageShort', { age: loan.player.age })}
              </Typography>
            </Stack>
            <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap', rowGap: 0.5, mt: 0.5 }}>
              <Typography variant="caption" color="text.secondary">
                <TeamNameLink teamId={loan.borrowingTeamId}>{loan.borrowingTeamName}</TeamNameLink>
              </Typography>
              <Chip size="small" variant="outlined" label={loan.borrowingDivisionName} />
              <Typography variant="caption" color="text.secondary">
                {t('team.loanedOut.returns', { season: loan.returnSeason })}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                {t('team.loanedOut.goals', { count: loan.goalsForBorrower })}
              </Typography>
            </Stack>
            {(loan.wageContributionPercent > 0 || loan.buyOptionFee > 0) && (
              <Stack direction="row" spacing={1} sx={{ flexWrap: 'wrap', rowGap: 0.5, mt: 0.5 }}>
                {loan.wageContributionPercent > 0 && (
                  <Chip size="small" label={t('team.loanedOut.wageShare', { percent: loan.wageContributionPercent })} />
                )}
                {loan.buyOptionFee > 0 && (
                  <Chip size="small" label={t('team.loanedOut.buyOption', { fee: formatCurrency(loan.buyOptionFee) })} />
                )}
              </Stack>
            )}
          </Paper>
        ))}
      </Stack>
    </Box>
  )
}
