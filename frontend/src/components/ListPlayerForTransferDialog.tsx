import { useEffect, useState } from 'react'
import Alert from '@mui/material/Alert'
import Button from '@mui/material/Button'
import Stack from '@mui/material/Stack'
import { useTranslation } from 'react-i18next'
import { useListPlayer, useTransferComparables } from '../api/hooks'
import type { Player } from '../api/types'
import { formatCurrency } from '../utils/formatCurrency'
import { NumberField } from './NumberField'
import { ResponsiveDialog } from './ResponsiveDialog'
import { TransferComparables } from './TransferComparables'

interface ListPlayerForTransferDialogProps {
  player: Player
  open: boolean
  onClose: () => void
}

/** Lists one of the caller's own players for transfer, straight from their player details page -
 * previously the only way there was the transfer market's own "List a player" form, which meant
 * navigating away and picking them back out of a dropdown. */
export function ListPlayerForTransferDialog({ player, open, onClose }: ListPlayerForTransferDialogProps) {
  const { t } = useTranslation()
  const listPlayer = useListPlayer()
  const comparables = useTransferComparables(player.id, open)
  const [askingPrice, setAskingPrice] = useState('')

  useEffect(() => {
    if (open) {
      setAskingPrice('')
      listPlayer.reset()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, player.id])

  const handleSubmit = () => {
    const price = Number(askingPrice)
    if (!price || price <= 0) return
    listPlayer.mutate({ playerId: player.id, askingPrice: price }, { onSuccess: onClose })
  }

  return (
    <ResponsiveDialog
      open={open}
      onClose={onClose}
      title={t('transfers.listPlayer.title')}
      subtitle={`${player.firstName} ${player.lastName}`}
      actions={
        <>
          <Button onClick={onClose}>{t('common.cancel')}</Button>
          <Button variant="contained" onClick={handleSubmit} disabled={!askingPrice || Number(askingPrice) <= 0 || listPlayer.isPending}>
            {t('transfers.list')}
          </Button>
        </>
      }
    >
      <Stack spacing={2}>
        <NumberField
          label={t('transfers.askingPrice')}
          value={askingPrice}
          onValueChange={setAskingPrice}
          currency
          fullWidth
          autoFocus
          helperText={t('transfers.listPlayer.hint', { wage: formatCurrency(player.wage) })}
        />
        <TransferComparables
          result={comparables.data}
          isLoading={comparables.isLoading}
          isError={comparables.isError}
          errorText={t('transfers.listPlayer.comparablesError')}
        />
        {listPlayer.isError && <Alert severity="error">{t('transfers.listPlayer.error')}</Alert>}
      </Stack>
    </ResponsiveDialog>
  )
}
