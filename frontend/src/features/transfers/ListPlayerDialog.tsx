import { useState } from 'react'
import Alert from '@mui/material/Alert'
import Button from '@mui/material/Button'
import FormControl from '@mui/material/FormControl'
import InputLabel from '@mui/material/InputLabel'
import MenuItem from '@mui/material/MenuItem'
import Select, { type SelectChangeEvent } from '@mui/material/Select'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useListPlayer, useTransferComparables } from '../../api/hooks'
import type { Player } from '../../api/types'
import { CountryFlag } from '../../components/CountryFlag'
import { NumberField } from '../../components/NumberField'
import { ResponsiveDialog } from '../../components/ResponsiveDialog'
import { TransferComparables } from '../../components/TransferComparables'
import { formatCurrency } from '../../utils/formatCurrency'

/** Dialog for putting one of your players on the market: pick who, name the price. Replaces the
 * always-visible form that pushed every listing 400px down the page. */
export function ListPlayerDialog({
  open,
  onClose,
  players,
}: {
  open: boolean
  onClose: () => void
  players: Player[]
}) {
  const { t } = useTranslation()
  const listPlayer = useListPlayer()
  const [selectedPlayerId, setSelectedPlayerId] = useState<number | ''>('')
  const [askingPrice, setAskingPrice] = useState('')
  const selected = players.find((p) => p.id === selectedPlayerId)
  const comparables = useTransferComparables(selectedPlayerId === '' ? undefined : selectedPlayerId, open)

  const handleList = () => {
    const price = Number(askingPrice)
    if (selectedPlayerId === '' || !price || price <= 0) {
      return
    }
    listPlayer.mutate(
      { playerId: selectedPlayerId, askingPrice: price },
      {
        onSuccess: () => {
          setSelectedPlayerId('')
          setAskingPrice('')
          onClose()
        },
      },
    )
  }

  return (
    <ResponsiveDialog
      open={open}
      onClose={onClose}
      title={t('transfers.listPlayer.title')}
      actions={
        <>
          <Button onClick={onClose}>{t('common.cancel')}</Button>
          <Button variant="contained" disabled={selectedPlayerId === '' || !askingPrice || listPlayer.isPending} onClick={handleList}>
            {t('transfers.list')}
          </Button>
        </>
      }
    >
      <Stack spacing={2}>
        <Typography variant="body2" color="text.secondary">
          {t('transfers.listDialog.hint')}
        </Typography>
        {listPlayer.isError && <Alert severity="error">{t('transfers.listPlayer.error')}</Alert>}
        <FormControl fullWidth>
          <InputLabel id="list-player-label">{t('transfers.listDialog.selectPlayer')}</InputLabel>
          <Select
            labelId="list-player-label"
            label={t('transfers.listDialog.selectPlayer')}
            value={selectedPlayerId}
            onChange={(event: SelectChangeEvent<number | ''>) => setSelectedPlayerId(event.target.value === '' ? '' : Number(event.target.value))}
          >
            {players.map((player) => (
              <MenuItem key={player.id} value={player.id}>
                <CountryFlag code={player.birthCountry} height={14} marginRight={0.5} />
                {player.firstName} {player.lastName}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        <NumberField
          label={t('transfers.askingPrice')}
          value={askingPrice}
          onValueChange={setAskingPrice}
          currency
          fullWidth
          helperText={selected ? t('transfers.listPlayer.hint', { wage: formatCurrency(selected.wage) }) : undefined}
        />
        {selected && (
          <TransferComparables
            result={comparables.data}
            isLoading={comparables.isLoading}
            isError={comparables.isError}
            errorText={t('transfers.listPlayer.comparablesError')}
          />
        )}
      </Stack>
    </ResponsiveDialog>
  )
}
