import { useState } from 'react'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Collapse from '@mui/material/Collapse'
import LinearProgress from '@mui/material/LinearProgress'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useTeamRivalries } from '../../api/hooks'
import { RIVALRY_MAX_SCORE, type TeamRivalry } from '../../api/types'
import { HeadToHeadCard } from '../../components/HeadToHeadCard'
import { RivalryChip } from '../../components/RivalryChip'
import { TeamBadge } from '../../components/TeamBadge'
import { TeamNameLink } from '../../components/TeamNameLink'

/** The team page's "Rivals" section (openspec rivalries "Rivalry visibility"): the club's top 3
 * pairs with a level, each with its Rival / Clásico chip, the score as a 0-40 bar and their
 * head-to-head. Hidden while the club has no rival. */
export function RivalsSection({ teamId, teamName }: { teamId: number; teamName: string }) {
  const { t } = useTranslation()
  const { data: rivals } = useTeamRivalries(String(teamId))
  if (!rivals || rivals.length === 0) {
    return null
  }
  return (
    <Stack spacing={1} component="section" aria-labelledby="team-rivals-heading">
      <Typography variant="h6" id="team-rivals-heading">
        {t('rivalries.title')}
      </Typography>
      {rivals.map((rival) => (
        <RivalRow key={rival.teamId} teamId={teamId} teamName={teamName} rival={rival} />
      ))}
    </Stack>
  )
}

function RivalRow({ teamId, teamName, rival }: { teamId: number; teamName: string; rival: TeamRivalry }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const score = Math.round(rival.score)
  return (
    <Paper variant="outlined" sx={{ p: 1.5 }} data-testid="rival-row">
      <Stack direction="row" spacing={1.25} sx={{ alignItems: 'center' }}>
        <TeamBadge teamId={rival.teamId} teamName={rival.teamName} size={32} />
        <Box sx={{ minWidth: 0, flex: 1 }}>
          <Typography variant="body2" sx={{ fontWeight: 600, overflowWrap: 'anywhere' }}>
            <TeamNameLink teamId={rival.teamId}>{rival.teamName}</TeamNameLink>
          </Typography>
          <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mt: 0.5 }}>
            <LinearProgress
              variant="determinate"
              value={Math.min(100, (rival.score / RIVALRY_MAX_SCORE) * 100)}
              color={rival.level >= 2 ? 'error' : 'warning'}
              sx={{ flex: 1, height: 6, borderRadius: 3 }}
              aria-label={t('rivalries.score', { score, max: RIVALRY_MAX_SCORE })}
            />
            <Typography variant="caption" color="text.secondary" sx={{ flexShrink: 0 }}>
              {t('rivalries.score', { score, max: RIVALRY_MAX_SCORE })}
            </Typography>
          </Stack>
        </Box>
        <RivalryChip level={rival.level} />
      </Stack>
      <Button size="small" onClick={() => setOpen((v) => !v)} aria-expanded={open} sx={{ mt: 0.5 }}>
        {t('rivalries.headToHead')}
      </Button>
      <Collapse in={open} unmountOnExit>
        <Box sx={{ mt: 1 }}>
          <HeadToHeadCard teamAId={teamId} teamAName={teamName} teamBId={rival.teamId} teamBName={rival.teamName} />
        </Box>
      </Collapse>
    </Paper>
  )
}
