import { useState } from 'react'
import Button from '@mui/material/Button'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import type { PreContractOffer } from '../../api/types'
import { ContractRenewalDialog } from '../../components/ContractRenewalDialog'
import { EmptyState } from '../../components/EmptyState'
import { TeamNameLink } from '../../components/TeamNameLink'
import { formatCurrency } from '../../utils/formatCurrency'
import { formatDateTime } from '../../utils/formatDate'

interface PreContractsSectionProps {
  received: PreContractOffer[]
  sent: PreContractOffer[]
  onWithdraw: (offer: PreContractOffer) => void
  withdrawPending: boolean
}

/** The pre-contracts part of the Offers tab (openspec/specs/pre-contracts): offers other clubs made
 * to your players in their last season - with when each player decides and the renewal wage that
 * keeps him - and the ones you made. */
export function PreContractsSection({ received, sent, onWithdraw, withdrawPending }: PreContractsSectionProps) {
  const { t } = useTranslation()
  return (
    <>
      <Typography variant="subtitle1" component="h2" sx={{ fontWeight: 600, mt: 3, mb: 1 }}>
        {t('transfers.preContract.onYourPlayers')}
      </Typography>
      {received.length === 0 ? (
        <EmptyState compact title={t('transfers.preContract.noneReceived')} />
      ) : (
        <Stack spacing={1}>
          {received.map((offer) => (
            <ReceivedPreContractCard key={offer.offerId} offer={offer} />
          ))}
        </Stack>
      )}

      <Typography variant="subtitle1" component="h2" sx={{ fontWeight: 600, mt: 3, mb: 1 }}>
        {t('transfers.preContract.yours')}
      </Typography>
      {sent.length === 0 ? (
        <EmptyState compact title={t('transfers.preContract.noneSent')} />
      ) : (
        <Stack spacing={1}>
          {sent.map((offer) => (
            <SentPreContractCard key={offer.offerId} offer={offer} onWithdraw={onWithdraw} withdrawPending={withdrawPending} />
          ))}
        </Stack>
      )}
    </>
  )
}

function termsLine(t: (key: string, options?: Record<string, unknown>) => string, offer: PreContractOffer) {
  return t('transfers.offers.termsLine', {
    wage: formatCurrency(offer.wage),
    years: t('team.contractDialog.yearsOption', { count: offer.years }),
  })
}

/** One pre-contract offer on your player: who offered what, when he decides and a Renew action
 * pre-filled with the wage that keeps him; once signed, that he leaves at the end of the season. */
export function ReceivedPreContractCard({ offer }: { offer: PreContractOffer }) {
  const { t } = useTranslation()
  const [renewOpen, setRenewOpen] = useState(false)
  const name = `${offer.player.firstName} ${offer.player.lastName}`
  return (
    <Paper variant="outlined" sx={{ p: 1.5 }}>
      <Stack spacing={0.5}>
        <Typography variant="body2" sx={{ fontWeight: 600 }}>
          {name}
        </Typography>
        <Typography variant="body2">
          <TeamNameLink teamId={offer.fromTeamId}>{offer.fromTeamName}</TeamNameLink> · {termsLine(t, offer)}
        </Typography>
        {offer.signedAt ? (
          <Typography variant="body2" color="warning.main">
            {t('transfers.preContract.signedAway', { team: offer.fromTeamName })}
          </Typography>
        ) : (
          <>
            {offer.decidesAt && (
              <Typography variant="caption" color="text.secondary">
                {t('transfers.preContract.decidesAt', { date: formatDateTime(offer.decidesAt) })}
              </Typography>
            )}
            {offer.keepWage !== undefined && offer.keepWage > 0 && (
              <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap', rowGap: 1 }}>
                <Typography variant="body2">{t('transfers.preContract.keepWage', { wage: formatCurrency(offer.keepWage) })}</Typography>
                <Button size="small" variant="contained" onClick={() => setRenewOpen(true)}>
                  {t('team.renewContract')}
                </Button>
              </Stack>
            )}
          </>
        )}
      </Stack>
      {renewOpen && <ContractRenewalDialog player={offer.player} open={renewOpen} onClose={() => setRenewOpen(false)} initialWage={offer.keepWage} />}
    </Paper>
  )
}

/** One pre-contract offer you made: pending (with when the player decides and a Withdraw action)
 * or signed (he joins at the end of the season). */
export function SentPreContractCard({
  offer,
  onWithdraw,
  withdrawPending,
}: {
  offer: PreContractOffer
  onWithdraw: (offer: PreContractOffer) => void
  withdrawPending: boolean
}) {
  const { t } = useTranslation()
  const name = `${offer.player.firstName} ${offer.player.lastName}`
  return (
    <Paper variant="outlined" sx={{ p: 1.5 }}>
      <Stack direction="row" spacing={1} sx={{ justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', rowGap: 1 }}>
        <Stack spacing={0.5} sx={{ minWidth: 0 }}>
          <Typography variant="body2" sx={{ fontWeight: 600 }}>
            {name} · <TeamNameLink teamId={offer.toTeamId}>{offer.toTeamName}</TeamNameLink>
          </Typography>
          <Typography variant="body2">{termsLine(t, offer)}</Typography>
          <Typography variant="caption" color={offer.signedAt ? 'success.main' : 'text.secondary'}>
            {offer.signedAt
              ? t('transfers.preContract.signed')
              : offer.decidesAt
                ? t('transfers.preContract.decidesAt', { date: formatDateTime(offer.decidesAt) })
                : ''}
          </Typography>
        </Stack>
        {!offer.signedAt && (
          <Button size="small" color="error" onClick={() => onWithdraw(offer)} disabled={withdrawPending}>
            {t('transfers.withdraw')}
          </Button>
        )}
      </Stack>
    </Paper>
  )
}
