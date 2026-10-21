import { Fragment } from 'react'
import { useState } from 'react'
import EmojiEventsTwoToneIcon from '@mui/icons-material/EmojiEventsTwoTone'
import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import ButtonBase from '@mui/material/ButtonBase'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Tab from '@mui/material/Tab'
import Tabs from '@mui/material/Tabs'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import { alpha, useTheme } from '@mui/material/styles'
import useMediaQuery from '@mui/material/useMediaQuery'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useCupDetails, useLatestCupId, useMe, useUserTeam } from '../../api/hooks'
import type { CupTie } from '../../api/types'
import { isPast, mightStillBeLive } from '../../api/useLiveMatch'
import { EmptyState } from '../../components/EmptyState'
import { PageHeader } from '../../components/PageHeader'
import { TeamBadge } from '../../components/TeamBadge'
import { NestedTeamNameLink } from '../../components/TeamNameLink'
import { formatDateTime } from '../../utils/formatDate'
import { translateCupName } from '../../utils/ledgerDescription'
import { localizedTeamName } from '../../utils/nationalTeamName'

export function CupPage() {
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const { data: latestCup, isLoading: cupIdLoading } = useLatestCupId(loggedUser?.country)
  const { data: cupDetails, isLoading: detailsLoading } = useCupDetails(latestCup?.cupId)

  if (cupIdLoading || detailsLoading) {
    return <Skeleton variant="rectangular" height={300} />
  }

  if (!cupDetails) {
    return (
      <Box>
        <PageHeader title={t('header.nav.cup')} />
        <EmptyState icon={<EmojiEventsTwoToneIcon fontSize="inherit" />} title={t('cup.noCupTitle')} text={t('cup.noCup')} />
      </Box>
    )
  }

  const { cup, ties } = cupDetails

  return (
    <Box>
      <PageHeader title={t('cup.titleWithSeason', { name: translateCupName(t, cup.name), season: cup.season })} />
      {cup.completed && (
        <Chip color="success" label={t('cup.champion', { name: cup.championTeamName })} sx={{ mb: 2 }} />
      )}

      <CupBracket ties={ties} />
    </Box>
  )
}

// A fixed per-tie row height/gap, in pixels, is what lets the bracket tree compute every box's
// vertical position (and the connector lines between them) with plain arithmetic instead of
// measuring rendered DOM nodes - see computeRoundCenters.
const BRACKET_ROW_HEIGHT = 56
const BRACKET_ROW_GAP = 20
const BRACKET_ROW_UNIT = BRACKET_ROW_HEIGHT + BRACKET_ROW_GAP
const BRACKET_COLUMN_WIDTH = 280
const BRACKET_CONNECTOR_WIDTH = 32

// CupBracket renders a cup's ties as a bracket tree - each round a column of tie cards, connected
// by lines to the pair of ties in the previous round that feed into them - rather than the
// previous stacked list of one Paper per round (issue #257: that list made it hard to see at a
// glance how a given team's run through the bracket connects round to round).
//
// Exported for InternationalCupPage and reused by CupPage above - a bracket is a bracket
// regardless of which competition it belongs to, same reasoning TieRow already relied on.
export function CupBracket({ ties }: { ties: CupTie[] }) {
  const { t } = useTranslation()
  const theme = useTheme()
  const isDesktop = useMediaQuery(theme.breakpoints.up('md'))
  const rounds = groupByRound(ties)
  const totalRounds = expectedTotalRounds(ties)
  const playIn = isFirstRoundPlayIn(ties)

  if (rounds.length === 0) return null

  // The 608px tree does not fit a phone (names get cut to 6-8 letters), so below `md` the same
  // ties are shown round by round instead.
  if (!isDesktop) {
    return <CupRoundsList rounds={rounds} totalRounds={totalRounds} playIn={playIn} />
  }

  const centers = computeRoundCenters(rounds.map(({ ties: roundTies }) => roundTies))
  // Round 1 always has the most ties (every later round is its winners, halved) - so sizing the
  // tree's height off round 1 alone is always enough to fit every later round's (fewer, more
  // widely spaced) boxes too.
  const treeHeight = (rounds[0].ties.length - 1) * BRACKET_ROW_UNIT + BRACKET_ROW_HEIGHT
  const connectorColor = theme.palette.divider

  return (
    <Box sx={{ overflowX: 'auto', WebkitOverflowScrolling: 'touch', pb: 1 }}>
      <Box sx={{ width: 'max-content' }}>
        <Stack direction="row">
          {rounds.map(({ round }, roundIdx) => (
            <Fragment key={round}>
              <Box sx={{ width: BRACKET_COLUMN_WIDTH, flexShrink: 0 }}>
                <Typography variant="subtitle2" align="center" sx={{ fontWeight: 600 }} gutterBottom>
                  {roundName(t, round, totalRounds, round === 1 && playIn)}
                </Typography>
              </Box>
              {roundIdx < rounds.length - 1 && <Box sx={{ width: BRACKET_CONNECTOR_WIDTH, flexShrink: 0 }} />}
            </Fragment>
          ))}
        </Stack>

        <Stack direction="row">
          {rounds.map(({ round, ties: roundTies }, roundIdx) => (
            <Fragment key={round}>
              <Box sx={{ width: BRACKET_COLUMN_WIDTH, flexShrink: 0, position: 'relative', height: treeHeight }}>
                {roundTies.map((tie, tieIdx) => (
                  <Box
                    key={tie.tieId}
                    sx={{
                      position: 'absolute',
                      top: centers[roundIdx][tieIdx] - BRACKET_ROW_HEIGHT / 2,
                      left: 4,
                      right: 4,
                      height: BRACKET_ROW_HEIGHT,
                    }}
                  >
                    <Paper variant="outlined" sx={{ height: '100%', overflow: 'hidden', display: 'flex', alignItems: 'center', px: 0.5 }}>
                      <TieRow tie={tie} />
                    </Paper>
                  </Box>
                ))}
              </Box>
              {roundIdx < rounds.length - 1 && (
                <BracketConnectors
                  height={treeHeight}
                  feederCenters={centers[roundIdx]}
                  childCenters={centers[roundIdx + 1]}
                  color={connectorColor}
                />
              )}
            </Fragment>
          ))}
        </Stack>
      </Box>
    </Box>
  )
}

// The phone view of a bracket: scrollable tabs, one per round (defaulting to the round in
// progress), listing that round's ties at full width with both team names in full. The user's own
// tie is highlighted and pinned to the top of its round.
function CupRoundsList({ rounds, totalRounds, playIn }: { rounds: { round: number; ties: CupTie[] }[]; totalRounds: number; playIn: boolean }) {
  const { t } = useTranslation()
  const { data: userTeam } = useUserTeam(true)
  const myTeamId = userTeam?.team.id
  const currentRound = rounds.find(({ ties: roundTies }) => roundTies.some((tie) => tie.awayTeamId !== null && !tie.played))?.round ?? rounds[rounds.length - 1].round
  const [selectedRound, setSelectedRound] = useState<number | null>(null)
  const active = rounds.find((r) => r.round === (selectedRound ?? currentRound)) ?? rounds[rounds.length - 1]

  const isMine = (tie: CupTie) => myTeamId !== undefined && (tie.homeTeamId === myTeamId || tie.awayTeamId === myTeamId)
  const ordered = [...active.ties].sort((a, b) => Number(isMine(b)) - Number(isMine(a)))

  return (
    <Box>
      <Tabs
        value={active.round}
        onChange={(_event, value: number) => setSelectedRound(value)}
        variant="scrollable"
        scrollButtons="auto"
        allowScrollButtonsMobile
        sx={{ mb: 2, borderBottom: 1, borderColor: 'divider' }}
      >
        {rounds.map(({ round }) => (
          <Tab key={round} value={round} label={roundName(t, round, totalRounds, round === 1 && playIn)} />
        ))}
      </Tabs>
      <Stack spacing={1.5}>
        {ordered.map((tie) => (
          <TieCard key={tie.tieId} tie={tie} mine={isMine(tie)} />
        ))}
      </Stack>
    </Box>
  )
}

function TieCard({ tie, mine }: { tie: CupTie; mine: boolean }) {
  const { t } = useTranslation()
  const theme = useTheme()
  const navigate = useNavigate()

  if (tie.awayTeamId === null) {
    return (
      <Paper variant="outlined" sx={{ p: 1.5 }}>
        <Typography variant="body2" color="text.secondary" sx={{ fontStyle: 'italic' }}>
          {t('cup.byeRests', { team: tie.homeTeamId ? localizedTeamName(t, tie.homeTeamName) : t('cup.tbd') })}
        </Typography>
      </Paper>
    )
  }

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
      <ButtonBase onClick={() => navigate(`/cup/ties/${tie.tieId}`)} sx={{ width: '100%', display: 'block', textAlign: 'left', p: 1.5 }}>
        {mine && <Chip size="small" color="primary" label={t('common.you')} sx={{ mb: 1 }} />}
        <Stack spacing={0.75}>
          {sides.map((side, index) => (
            <Stack key={index} direction="row" spacing={1} sx={{ alignItems: 'center' }}>
              {side.id ? <TeamBadge teamId={side.id} teamName={side.name} size={24} /> : <Box sx={{ width: 24 }} />}
              <Typography
                variant="body2"
                sx={{ flexGrow: 1, overflowWrap: 'anywhere', fontWeight: tie.wentToPenalties && tie.winnerTeamId === side.id ? 700 : 500 }}
              >
                {side.id ? localizedTeamName(t, side.name) : t('cup.tbd')}
              </Typography>
              {showScore && (
                <Typography variant="body2" sx={{ fontWeight: 700, fontVariantNumeric: 'tabular-nums' }}>
                  {side.goals}
                </Typography>
              )}
            </Stack>
          ))}
        </Stack>
        <Stack direction="row" spacing={1} sx={{ mt: 1, alignItems: 'center' }}>
          {stillLive ? (
            <Chip label={t('match.live')} color="error" size="small" sx={{ fontWeight: 700 }} />
          ) : (
            !showScore && (
              <Typography variant="caption" color="text.secondary">
                {formatDateTime(tie.matchDate ?? '')}
              </Typography>
            )
          )}
          {!stillLive && tie.wentToPenalties && (
            <Typography variant="caption" color="text.secondary">
              {penaltyWinnerName ? t('cup.pensWinner', { team: localizedTeamName(t, penaltyWinnerName) }) : t('cup.pens')}
            </Typography>
          )}
        </Stack>
      </ButtonBase>
    </Paper>
  )
}

// computeRoundCenters derives each tie box's vertical center (in pixels) bottom-up: round 1's
// boxes are evenly spaced one BRACKET_ROW_UNIT apart, and every later round's tie k is centered on
// the midpoint between the two previous-round ties (at indices 2k and 2k+1) that feed into it -
// the standard bracket-tree layout, which also happens to put every child exactly on the straight
// line between its two feeders, so a single vertical connector segment reaches all three.
// A lone feeder (round sizes that don't halve evenly - see backend engine.pairSequentially's
// legacy odd-round-bye note) just passes its own center up unchanged instead of averaging.
function computeRoundCenters(roundsTies: CupTie[][]): number[][] {
  const centers: number[][] = [roundsTies[0].map((_, i) => i * BRACKET_ROW_UNIT + BRACKET_ROW_HEIGHT / 2)]
  for (let r = 1; r < roundsTies.length; r++) {
    const prev = centers[r - 1]
    centers.push(
      roundsTies[r].map((_, i) => {
        const a = prev[2 * i]
        const b = prev[2 * i + 1]
        if (b === undefined) return a
        return (a + b) / 2
      }),
    )
  }
  return centers
}

// BracketConnectors draws the lines between one round's tie boxes and the next round's: each
// child center sits on the vertical segment joining its two feeders (see computeRoundCenters), so
// that segment plus a horizontal stub out of it to the child box is enough - no per-feeder stub is
// needed on the child side since all three points are already collinear from the feeders' corner.
function BracketConnectors({
  height,
  feederCenters,
  childCenters,
  color,
}: {
  height: number
  feederCenters: number[]
  childCenters: number[]
  color: string
}) {
  const midX = BRACKET_CONNECTOR_WIDTH / 2
  return (
    <Box sx={{ width: BRACKET_CONNECTOR_WIDTH, flexShrink: 0 }}>
      <svg width={BRACKET_CONNECTOR_WIDTH} height={height} style={{ display: 'block' }}>
        {childCenters.map((childCenter, i) => {
          const a = feederCenters[2 * i]
          const b = feederCenters[2 * i + 1]
          if (b === undefined) {
            return <line key={i} x1={0} y1={a} x2={BRACKET_CONNECTOR_WIDTH} y2={childCenter} stroke={color} strokeWidth={2} />
          }
          return (
            <g key={i}>
              <path d={`M0,${a} H${midX} V${b} H0`} fill="none" stroke={color} strokeWidth={2} />
              <line x1={midX} y1={childCenter} x2={BRACKET_CONNECTOR_WIDTH} y2={childCenter} stroke={color} strokeWidth={2} />
            </g>
          )
        })}
      </svg>
    </Box>
  )
}

// A tie decided by a penalty shootout ends level on the scoreboard (0-0, 1-1, ...), so unlike a
// normally-decided tie, the score alone can't tell a reader who actually advances - only
// tie.winnerTeamId (set by the coin-flip shootout in backend engine.simulateCupTie) says that.
// Returns undefined for a tie that wasn't decided on penalties, or isn't played/resolved yet, so
// callers can fall back to the plain "(pens)" caption (issue: bracket/tie page said penalties
// happened but never said who won).
export function cupTiePenaltyWinnerName(tie: CupTie): string | undefined {
  if (!tie.wentToPenalties || tie.winnerTeamId === null) return undefined
  return tie.winnerTeamId === tie.homeTeamId ? tie.homeTeamName : tie.awayTeamName
}

// Exported for InternationalCupPage - identical bracket rendering, just keyed by region instead
// of country to find its cup id (see useLatestInternationalCupId).
export function TieRow({ tie }: { tie: CupTie }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const isBye = tie.awayTeamId === null

  // A bye isn't a match - there's no opponent, no score and nothing to click into - so rather than
  // squeezing "Bye" into the score column of an otherwise ordinary-looking match row (which reads
  // as a 0-0 result against nobody), it gets its own distinct, non-clickable "rests this round"
  // row with just the one team's name.
  if (isBye) {
    return (
      <Box sx={{ py: 0.5, px: 1, width: '100%', minWidth: 0 }}>
        <Typography variant="body2" color="text.secondary" noWrap sx={{ fontStyle: 'italic' }}>
          {t('cup.byeRests', { team: tie.homeTeamId ? localizedTeamName(t, tie.homeTeamName) : t('cup.tbd') })}
        </Typography>
      </Box>
    )
  }

  // See DivisionDetailsPage's identical mightStillBeLive check - a tie flips played=true the
  // instant it's simulated, well before its result finishes broadcasting live.
  const stillLive = tie.played && mightStillBeLive(tie.matchDate ?? '')
  const penaltyWinnerName = cupTiePenaltyWinnerName(tie)
  const homeWonPenalties = tie.wentToPenalties && tie.winnerTeamId === tie.homeTeamId
  const awayWonPenalties = tie.wentToPenalties && tie.winnerTeamId === tie.awayTeamId
  // This bracket is shared by regular club cups and the World Cup knockout stage alike (see
  // NationalTeamsPage's WorldCupSection) - localizedTeamName is a no-op for an ordinary club name,
  // so it's always safe to apply here regardless of which one a given tie belongs to.

  return (
    <ButtonBase onClick={() => navigate(`/cup/ties/${tie.tieId}`)} sx={{ width: '100%', justifyContent: 'stretch', borderRadius: 1 }}>
      <Stack direction="row" spacing={1} sx={{ py: 0.5, alignItems: 'center', width: '100%' }}>
        <Typography variant="body2" noWrap sx={{ flex: 1, minWidth: 0, textAlign: 'left' }}>
          {tie.homeTeamId ? (
            <NestedTeamNameLink teamId={tie.homeTeamId} sx={{ fontWeight: homeWonPenalties ? 700 : undefined }}>
              {localizedTeamName(t, tie.homeTeamName)}
            </NestedTeamNameLink>
          ) : (
            t('cup.tbd')
          )}
        </Typography>
        {stillLive ? (
          <Chip label={t('match.live')} color="error" size="small" sx={{ fontWeight: 700 }} />
        ) : (
          <Typography variant="body2" sx={{ minWidth: 80, textAlign: 'center' }}>
            {tie.played && isPast(tie.matchDate ?? '') ? `${tie.homeGoals} - ${tie.awayGoals}` : formatDateTime(tie.matchDate ?? '')}
          </Typography>
        )}
        <Typography variant="body2" noWrap sx={{ flex: 1, minWidth: 0, textAlign: 'right' }}>
          {tie.awayTeamId ? (
            <NestedTeamNameLink teamId={tie.awayTeamId} sx={{ fontWeight: awayWonPenalties ? 700 : undefined }}>
              {localizedTeamName(t, tie.awayTeamName)}
            </NestedTeamNameLink>
          ) : (
            t('cup.tbd')
          )}
        </Typography>
        {!stillLive && tie.wentToPenalties && (
          <Tooltip title={penaltyWinnerName ? t('cup.pensWinner', { team: localizedTeamName(t, penaltyWinnerName) }) : ''}>
            <Typography variant="caption" color="text.secondary">
              {t('cup.pens')}
            </Typography>
          </Tooltip>
        )}
      </Stack>
    </ButtonBase>
  )
}

export function groupByRound(ties: CupTie[]): { round: number; ties: CupTie[] }[] {
  const byRound = new Map<number, CupTie[]>()
  for (const tie of ties) {
    const list = byRound.get(tie.round) ?? []
    list.push(tie)
    byRound.set(tie.round, list)
  }
  return Array.from(byRound.entries())
    .sort(([a], [b]) => a - b)
    .map(([round, roundTies]) => ({
      round,
      ties: [...roundTies].sort((a, b) => a.slotNumber - b.slotNumber),
    }))
}

export function roundName(
  t: (key: string, options?: Record<string, unknown>) => string,
  round: number,
  totalRounds: number,
  isPlayIn = false,
): string {
  // A play-in round only ever happens as round 1 (see backend engine.BuildFirstRoundPairings) -
  // it isn't really this bracket's "round 1" so much as a preliminary trim down to a clean
  // power-of-two field, so it gets its own distinct label instead of counting toward
  // quarterfinal/semifinal/final the way an ordinary round does.
  if (isPlayIn) return t('cup.playInRound')
  const roundsFromEnd = totalRounds - round
  if (roundsFromEnd === 0) return t('cup.final')
  if (roundsFromEnd === 1) return t('cup.semifinal')
  if (roundsFromEnd === 2) return t('cup.quarterfinal')
  return t('cup.round', { round })
}

// expectedTotalRounds derives how many rounds this bracket needs to go all the way down to a
// single champion, from the first round's entrant count alone - rather than from how many rounds
// have been generated/played so far (rounds.length), which is wrong for any field that isn't a
// power of two. backend engine.pairSequentially pairs entrants two at a time and gives the
// leftover a bye whenever a round's entrant count is odd - not just in round 1 - so a field like 9
// teams needs four rounds (9 -> 5 -> 3 -> 2 -> 1), each one of those reductions halving-and-
// rounding-up the previous count. That reduction always takes exactly ceil(log2(entrants)) steps.
// Using rounds.length instead (the previous behavior) mislabeled the next-to-last generated round
// as "Final" - complete with a bye team advancing "in the final" - the moment it merely happened
// to be the last round generated so far, well before the bracket had actually narrowed to two
// teams.
export function expectedTotalRounds(ties: CupTie[]): number {
  if (ties.length === 0) return 0
  const entrants = firstRoundEntrantCount(ties)
  return entrants <= 1 ? 0 : Math.ceil(Math.log2(entrants))
}

function firstRoundEntrantCount(ties: CupTie[]): number {
  if (ties.length === 0) return 0
  const firstRound = Math.min(...ties.map((tie) => tie.round))
  return ties
    .filter((tie) => tie.round === firstRound)
    .reduce((count, tie) => count + (tie.homeTeamId !== null ? 1 : 0) + (tie.awayTeamId !== null ? 1 : 0), 0)
}

// isFirstRoundPlayIn tells whether this bracket's round 1 doubles as a play-in/preliminary round
// (see backend engine.BuildFirstRoundPairings): that only happens when round 1's entrant count
// isn't already an exact power of two, in which case several entrants there get a bye solely to
// trim the field down to one, rather than round 1 being a normal round that merely happens to
// have a single leftover bye.
export function isFirstRoundPlayIn(ties: CupTie[]): boolean {
  const entrants = firstRoundEntrantCount(ties)
  return entrants > 1 && !isPowerOfTwo(entrants)
}

function isPowerOfTwo(n: number): boolean {
  return n > 0 && (n & (n - 1)) === 0
}
