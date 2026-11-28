import { useState } from 'react'
import LeaderboardOutlinedIcon from '@mui/icons-material/LeaderboardOutlined'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import ToggleButton from '@mui/material/ToggleButton'
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useLeagueLeaders } from '../../api/hooks'
import type { LeadersCategory } from '../../api/types'
import { CountryFlag } from '../../components/CountryFlag'
import { EmptyState } from '../../components/EmptyState'
import { PlayerNameLink } from '../../components/PlayerNameLink'
import { TeamNameLink } from '../../components/TeamNameLink'
import { leaderValue } from './leaders'

const CATEGORIES: LeadersCategory[] = ['goals', 'rating', 'minutes', 'cards']
const RANK_COLORS = ['#D4AF37', '#A8A9AD', '#B08D57'] as const

// The league's leaders of one season (openspec player-stats "League leaders"): top scorers by
// default, or the best average rating, most minutes and most cards. `season` undefined is the
// current season.
export function LeagueLeadersCard({ leagueId, season, limit = 5 }: { leagueId: string | undefined; season?: number; limit?: number }) {
  const { t } = useTranslation()
  const [category, setCategory] = useState<LeadersCategory>('goals')
  const [showAll, setShowAll] = useState(false)
  const { data, isLoading } = useLeagueLeaders(leagueId, category, season)
  // A list for another category (kept while the new one loads) or an unexpected response is
  // not shown.
  const leaders = data?.category === category && Array.isArray(data.leaders) ? data.leaders : isLoading ? undefined : []
  const visible = leaders ? (showAll ? leaders : leaders.slice(0, limit)) : []

  return (
    <Paper variant="outlined">
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center', p: 2, pb: 1 }}>
        <LeaderboardOutlinedIcon color="primary" fontSize="small" />
        <Typography variant="subtitle1" component="h2" sx={{ fontWeight: 700 }}>
          {t('stats.leaders.title')}
        </Typography>
      </Stack>
      <Box sx={{ px: 2, pb: 1 }}>
        <ToggleButtonGroup
          exclusive
          size="small"
          fullWidth
          value={category}
          onChange={(_event, value: LeadersCategory | null) => value && setCategory(value)}
          aria-label={t('stats.leaders.title')}
        >
          {CATEGORIES.map((value) => (
            <ToggleButton key={value} value={value} sx={{ px: 0.5 }}>
              {t(`stats.leaders.categories.${value}`)}
            </ToggleButton>
          ))}
        </ToggleButtonGroup>
      </Box>
      {isLoading || !leaders ? (
        <Skeleton variant="rectangular" height={100} sx={{ m: 2, borderRadius: 1 }} />
      ) : leaders.length === 0 ? (
        <EmptyState compact title={t(`stats.leaders.empty.${category}`)} />
      ) : (
        <>
          <Stack divider={<Box sx={{ borderBottom: '1px solid', borderColor: 'divider' }} />}>
            {visible.map((leader, index) => {
              const rankColor = RANK_COLORS[index]
              return (
                <Stack key={leader.playerId} direction="row" spacing={1.5} sx={{ alignItems: 'center', px: 2, py: 1 }}>
                  <Box
                    sx={{
                      width: 22,
                      height: 22,
                      flexShrink: 0,
                      borderRadius: '50%',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      typography: 'chipLabel',
                      fontWeight: 700,
                      bgcolor: rankColor ?? 'action.selected',
                      color: rankColor ? '#fff' : 'text.secondary',
                    }}
                  >
                    {index + 1}
                  </Box>
                  <Box sx={{ flexGrow: 1, minWidth: 0 }}>
                    <Typography variant="body2" noWrap sx={{ fontWeight: 600 }}>
                      <CountryFlag code={leader.birthCountry} height={14} marginRight={0.5} />
                      <PlayerNameLink playerId={leader.playerId}>
                        {leader.firstName} {leader.lastName}
                      </PlayerNameLink>
                    </Typography>
                    <Typography variant="caption" color="text.secondary" noWrap sx={{ display: 'block' }}>
                      <TeamNameLink teamId={leader.teamId}>{leader.teamName}</TeamNameLink>
                      {' · '}
                      {t('stats.leaders.appearances', { count: leader.appearances })}
                    </Typography>
                  </Box>
                  <Typography variant="body2" sx={{ fontWeight: 700, whiteSpace: 'nowrap' }}>
                    {leaderValue(category, leader)}
                  </Typography>
                </Stack>
              )
            })}
          </Stack>
          {leaders.length > limit && (
            <Box sx={{ p: 1, textAlign: 'center' }}>
              <Button onClick={() => setShowAll((current) => !current)}>
                {showAll ? t('common.viewLess') : t('division.topScorers.viewAll', { count: leaders.length })}
              </Button>
            </Box>
          )}
          {category === 'rating' && (
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', px: 2, pb: 1.5 }}>
              {t('stats.leaders.ratingNote')}
            </Typography>
          )}
        </>
      )}
    </Paper>
  )
}
