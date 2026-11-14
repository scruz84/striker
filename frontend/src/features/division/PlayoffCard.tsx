import SwapVertOutlinedIcon from '@mui/icons-material/SwapVertOutlined'
import Box from '@mui/material/Box'
import ButtonBase from '@mui/material/ButtonBase'
import Chip from '@mui/material/Chip'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { alpha, useTheme } from '@mui/material/styles'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import type { CupTie } from '../../api/types'
import { isPast, mightStillBeLive } from '../../api/useLiveMatch'
import { TeamBadge } from '../../components/TeamBadge'
import { TeamNameWithFlag } from '../../components/TeamNameWithFlag'
import { formatDateTime } from '../../utils/formatDate'
import { localizedTeamName } from '../../utils/nationalTeamName'
import { cupTiePenaltyWinnerName } from '../cup/CupPage'

// PlayoffCard lists the current season's promotion playoff ties with a side in this league (see
// openspec/specs/league-page "Playoff section"): both teams (the upper-tier side at home), the
// kickoff, and the score once the tie's broadcast is over, with a penalties mark. Each row opens the
// tie's match page.
export function PlayoffCard({ ties, myTeamId }: { ties: CupTie[]; myTeamId: number | undefined }) {
  const { t } = useTranslation()
  return (
    <Paper variant="outlined" component="section" aria-labelledby="division-playoff-title">
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center', p: 2, pb: 1 }}>
        <SwapVertOutlinedIcon color="warning" fontSize="small" />
        <Typography id="division-playoff-title" variant="subtitle1" component="h2" sx={{ fontWeight: 700 }}>
          {t('division.playoff.title')}
        </Typography>
      </Stack>
      <Typography variant="body2" color="text.secondary" sx={{ px: 2, pb: 1 }}>
        {t('division.playoff.description')}
      </Typography>
      <Stack spacing={1} sx={{ px: 2, pb: 2 }}>
        {ties.map((tie) => (
          <PlayoffTieRow key={tie.tieId} tie={tie} mine={myTeamId !== undefined && (tie.homeTeamId === myTeamId || tie.awayTeamId === myTeamId)} />
        ))}
      </Stack>
    </Paper>
  )
}

function PlayoffTieRow({ tie, mine }: { tie: CupTie; mine: boolean }) {
  const { t } = useTranslation()
  const theme = useTheme()
  const navigate = useNavigate()
  const stillLive = tie.played && mightStillBeLive(tie.matchDate ?? '')
  const showScore = !stillLive && tie.played && isPast(tie.matchDate ?? '')
  const penaltyWinnerName = cupTiePenaltyWinnerName(tie)
  const sides = [
    { id: tie.homeTeamId, name: tie.homeTeamName, goals: tie.homeGoals },
    { id: tie.awayTeamId, name: tie.awayTeamName, goals: tie.awayGoals },
  ]

  return (
    <Paper
      variant="outlined"
      sx={{ borderColor: mine ? 'primary.main' : undefined, borderWidth: mine ? 2 : 1, bgcolor: mine ? alpha(theme.palette.primary.main, 0.06) : undefined }}
    >
      <ButtonBase
        onClick={() => navigate(`/cup/ties/${tie.tieId}`)}
        aria-label={t('division.playoff.openTie', { home: localizedTeamName(t, tie.homeTeamName), away: localizedTeamName(t, tie.awayTeamName) })}
        sx={{ width: '100%', display: 'block', textAlign: 'left', p: 1.5 }}
      >
        <Stack spacing={0.75}>
          {sides.map((side, index) => (
            <Stack key={index} direction="row" spacing={1} sx={{ alignItems: 'center', minWidth: 0 }}>
              {side.id ? <TeamBadge teamId={side.id} teamName={side.name} size={24} /> : <Box sx={{ width: 24 }} />}
              <Typography
                variant="body2"
                sx={{ flexGrow: 1, minWidth: 0, overflowWrap: 'anywhere', fontWeight: showScore && tie.winnerTeamId === side.id ? 700 : 500 }}
              >
                <TeamNameWithFlag rawName={side.name} />
              </Typography>
              {showScore && (
                <Typography variant="body2" sx={{ fontWeight: 700, fontVariantNumeric: 'tabular-nums' }}>
                  {side.goals}
                </Typography>
              )}
            </Stack>
          ))}
        </Stack>
        <Stack direction="row" spacing={1} sx={{ mt: 1, alignItems: 'center', flexWrap: 'wrap', rowGap: 0.5 }}>
          {mine && <Chip size="small" color="primary" label={t('common.you')} />}
          {stillLive ? (
            <Chip label={t('match.live')} color="error" size="small" sx={{ fontWeight: 700 }} />
          ) : (
            !showScore && (
              <Typography variant="caption" color="text.secondary">
                {formatDateTime(tie.matchDate ?? '')}
              </Typography>
            )
          )}
          {showScore && tie.wentToPenalties && (
            <Typography variant="caption" color="text.secondary">
              {penaltyWinnerName ? t('cup.pensWinner', { team: localizedTeamName(t, penaltyWinnerName) }) : t('cup.pens')}
            </Typography>
          )}
        </Stack>
      </ButtonBase>
    </Paper>
  )
}
