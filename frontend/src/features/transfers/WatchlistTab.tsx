import { useState } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useAcceptLoan, useBuyListing, useMe, useUserTeam, useWatchlist } from '../../api/hooks'
import { MAX_WATCHLIST_SIZE, type WatchlistEntry } from '../../api/types'
import { CartoonAvatar } from '../../components/avatar/CartoonAvatar'
import { useConfirm } from '../../components/confirmDialogContext'
import { CountryFlag } from '../../components/CountryFlag'
import { FreeAgentSigningDialog } from '../../components/FreeAgentSigningDialog'
import { PlayerNameLink } from '../../components/PlayerNameLink'
import { TeamNameLink } from '../../components/TeamNameLink'
import { TransferOfferDialog } from '../../components/TransferOfferDialog'
import { WatchToggle } from '../../components/WatchToggle'
import { formatCurrency } from '../../utils/formatCurrency'

/** The transfer market's Watchlist tab (openspec player-watchlist "Watchlist view"): every followed
 * player with where he stands - his club, or on the transfer or loan list, or a free agent - and
 * the one action that status allows. */
export function WatchlistTab() {
  const { t } = useTranslation()
  const { data: me } = useMe()
  const { data: userTeam } = useUserTeam(!!me)
  const { data: watchlist, isLoading } = useWatchlist(!!userTeam)

  if (!userTeam) {
    return <Alert severity="info">{t('watchlist.needTeam')}</Alert>
  }
  if (isLoading || !watchlist) {
    return <Skeleton variant="rectangular" height={300} />
  }
  if (watchlist.length === 0) {
    return <Alert severity="info">{t('watchlist.empty')}</Alert>
  }
  return (
    <Box>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 1.5 }}>
        {t('watchlist.count', { count: watchlist.length, max: MAX_WATCHLIST_SIZE })}
      </Typography>
      <Stack spacing={1}>
        {watchlist.map((entry) => (
          <WatchlistRow key={entry.player.id} entry={entry} />
        ))}
      </Stack>
    </Box>
  )
}

function WatchlistRow({ entry }: { entry: WatchlistEntry }) {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const buyListing = useBuyListing()
  const acceptLoan = useAcceptLoan()
  const [offerOpen, setOfferOpen] = useState(false)
  const [signOpen, setSignOpen] = useState(false)
  const { player, transferListing, loanListing } = entry
  const name = `${player.firstName} ${player.lastName}`
  const freeAgent = player.teamId === 0

  let status: string
  let action: React.ReactNode
  if (transferListing) {
    status = t('watchlist.status.transferListed', { price: formatCurrency(transferListing.askingPrice) })
    action = (
      <Button
        variant="contained"
        disabled={buyListing.isPending}
        onClick={async () => {
          const ok = await confirm({
            message: t('transfers.confirmBuy', { name, price: formatCurrency(transferListing.askingPrice) }),
            confirmLabel: t('transfers.buy'),
            destructive: true,
          })
          if (ok) buyListing.mutate(transferListing.listingId)
        }}
      >
        {t('transfers.buy')}
      </Button>
    )
  } else if (loanListing) {
    status = t('watchlist.status.loanListed', { count: loanListing.durationSeasons })
    action = (
      <Button
        variant="contained"
        disabled={acceptLoan.isPending}
        onClick={async () => {
          const message = loanListing.loanFee
            ? t('transfers.confirmAcceptLoanWithFee', { name, seasons: loanListing.durationSeasons, fee: formatCurrency(loanListing.loanFee) })
            : t('transfers.confirmAcceptLoan', { name, seasons: loanListing.durationSeasons })
          const ok = await confirm({ message, confirmLabel: t('transfers.accept'), destructive: !!loanListing.loanFee })
          if (ok) acceptLoan.mutate(loanListing.listingId)
        }}
      >
        {t('watchlist.takeOnLoan')}
      </Button>
    )
  } else if (freeAgent) {
    status = t('watchlist.status.freeAgent')
    action = (
      <Button variant="contained" onClick={() => setSignOpen(true)}>
        {t('transfers.freeAgents.sign')}
      </Button>
    )
  } else {
    status = t('watchlist.status.atClub')
    action = (
      <Button variant="outlined" onClick={() => setOfferOpen(true)}>
        {t('team.makeOffer')}
      </Button>
    )
  }

  return (
    <Paper variant="outlined" sx={{ p: 1.5 }} data-testid="watchlist-row">
      <Stack direction="row" spacing={1.25} sx={{ alignItems: 'center' }}>
        <CartoonAvatar seed={player.id} birthCountry={player.birthCountry} size={40} />
        <Box sx={{ minWidth: 0, flex: 1 }}>
          <Typography variant="body2" sx={{ fontWeight: 600, overflowWrap: 'anywhere' }}>
            <CountryFlag code={player.birthCountry} height={14} marginRight={0.5} />
            <PlayerNameLink playerId={player.id}>{name}</PlayerNameLink>
          </Typography>
          <Typography variant="caption" color="text.secondary" component="div">
            {t('transfers.ageLabel', { age: player.age })}
            {player.averageSkill !== undefined && ` · ${t('team.averageSkill')} ${player.averageSkill}`}
            {' · '}
            {freeAgent || !entry.teamName ? t('search.freeAgent') : <TeamNameLink teamId={player.teamId}>{entry.teamName}</TeamNameLink>}
          </Typography>
          <Typography variant="body2" sx={{ mt: 0.25 }}>
            {status}
          </Typography>
        </Box>
        <WatchToggle player={player} />
      </Stack>
      <Stack direction="row" spacing={1} sx={{ justifyContent: 'flex-end', mt: 1 }}>
        {action}
      </Stack>
      {offerOpen && <TransferOfferDialog player={player} open={offerOpen} onClose={() => setOfferOpen(false)} />}
      {signOpen && <FreeAgentSigningDialog player={player} open={signOpen} onClose={() => setSignOpen(false)} />}
    </Paper>
  )
}
