import { useEffect, useState } from 'react'
import Alert from '@mui/material/Alert'
import Button from '@mui/material/Button'
import Dialog from '@mui/material/Dialog'
import DialogActions from '@mui/material/DialogActions'
import DialogContent from '@mui/material/DialogContent'
import DialogTitle from '@mui/material/DialogTitle'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import { useTranslation } from 'react-i18next'
import { useListPlayer } from '../api/hooks'
import type { Player } from '../api/types'

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
    <Dialog open={open} onClose={onClose} maxWidth="xs" fullWidth>
      <DialogTitle>{t('transfers.listPlayer.title')}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ mt: 1 }}>
          <TextField
            label={t('transfers.askingPrice')}
            type="number"
            value={askingPrice}
            onChange={(e) => setAskingPrice(e.target.value)}
            fullWidth
            size="small"
            autoFocus
          />
          {listPlayer.isError && <Alert severity="error">{t('transfers.listPlayer.error')}</Alert>}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t('common.cancel')}</Button>
        <Button variant="contained" onClick={handleSubmit} disabled={!askingPrice || Number(askingPrice) <= 0 || listPlayer.isPending}>
          {t('transfers.list')}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
