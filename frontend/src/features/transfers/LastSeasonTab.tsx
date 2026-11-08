import { useState } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useMe, usePreContractMarket, useUserTeam } from '../../api/hooks'
import { POSITION_SHORT_LABEL_KEYS, type PreContractMarketEntry } from '../../api/types'
import { CartoonAvatar } from '../../components/avatar/CartoonAvatar'
import { PlayerNameLink } from '../../components/PlayerNameLink'
import { PreContractDialog } from '../../components/PreContractDialog'
import { TeamNameLink } from '../../components/TeamNameLink'
import { TransferOfferDialog } from '../../components/TransferOfferDialog'
import { WatchToggle } from '../../components/WatchToggle'
import { formatCurrency } from '../../utils/formatCurrency'
import { ContractMeta } from './marketUi'

/** The transfer market's "Last season" tab (openspec/specs/pre-contracts "Pre-contract market"):
 * the best players of other clubs in the last season of their contract, with whether a
 * pre-contract offer is open yet (their league halfway through) and the actions it allows. */
export function LastSeasonTab() {
  const { t } = useTranslation()
  const { data: me } = useMe()
  const { data: userTeam } = useUserTeam(!!me)
  const { data: entries, isLoading } = usePreContractMarket(!!userTeam)

  if (!userTeam) {
    return <Alert severity="info">{t('watchlist.needTeam')}</Alert>
  }
  if (isLoading || !entries) {
    return <Skeleton variant="rectangular" height={300} />
  }
  return (
    <Box>
      <Alert severity="info" sx={{ mb: 1.5 }}>
        {t('transfers.lastSeason.intro')}
      </Alert>
      {entries.length === 0 ? (
        <Alert severity="info">{t('transfers.lastSeason.empty')}</Alert>
      ) : (
        <Stack spacing={1}>
          {entries.map((entry) => (
            <LastSeasonRow key={entry.player.id} entry={entry} />
          ))}
        </Stack>
      )}
    </Box>
  )
}

export function LastSeasonRow({ entry }: { entry: PreContractMarketEntry }) {
  const { t } = useTranslation()
  const [preContractOpen, setPreContractOpen] = useState(false)
  const [offerOpen, setOfferOpen] = useState(false)
  const { player } = entry
  return (
    <Paper variant="outlined" sx={{ p: 1.5 }} data-testid="last-season-row">
      <Stack direction="row" spacing={1.25} sx={{ alignItems: 'center' }}>
        <CartoonAvatar seed={player.id} birthCountry={player.birthCountry} size={40} />
        <Box sx={{ minWidth: 0, flex: 1 }}>
          <Typography variant="body2" sx={{ fontWeight: 600, overflowWrap: 'anywhere' }}>
            <PlayerNameLink playerId={player.id}>
              {player.firstName} {player.lastName}
            </PlayerNameLink>
          </Typography>
          <Typography variant="caption" color="text.secondary" component="div">
            {t(POSITION_SHORT_LABEL_KEYS[entry.bestPosition])} · {t('lineup.ageShort', { age: player.age })} ·{' '}
            {t('transfers.lastSeason.averageSkill', { skill: player.averageSkill ?? 0 })} ·{' '}
            <TeamNameLink teamId={player.teamId}>{entry.teamName}</TeamNameLink>
          </Typography>
          <Typography variant="caption" color="text.secondary" component="div">
            {t('transfers.salaryPerMatch', { amount: formatCurrency(player.wage) })}
          </Typography>
          <ContractMeta player={player} />
          {!entry.open && (
            <Typography variant="caption" color="text.secondary" component="div">
              {t('transfers.lastSeason.notOpen')}
            </Typography>
          )}
        </Box>
        <WatchToggle player={player} size="small" />
      </Stack>
      <Stack direction="row" spacing={1} sx={{ justifyContent: 'flex-end', mt: 1 }}>
        <Button variant="outlined" onClick={() => setOfferOpen(true)}>
          {t('team.makeOffer')}
        </Button>
        <Button variant="contained" disabled={!entry.open || entry.offered} onClick={() => setPreContractOpen(true)}>
          {entry.offered ? t('transfers.lastSeason.offered') : t('transfers.preContract.button')}
        </Button>
      </Stack>
      {preContractOpen && <PreContractDialog player={player} open={preContractOpen} onClose={() => setPreContractOpen(false)} />}
      {offerOpen && <TransferOfferDialog player={player} open={offerOpen} onClose={() => setOfferOpen(false)} />}
    </Paper>
  )
}
