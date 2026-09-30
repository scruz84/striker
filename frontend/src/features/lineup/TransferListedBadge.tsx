import Box from '@mui/material/Box'
import Tooltip from '@mui/material/Tooltip'
import SellIcon from '@mui/icons-material/Sell'
import { useTranslation } from 'react-i18next'

/** Flags a player currently listed on the transfer market, the same "small icon next to the name"
 * treatment UnavailabilityBadge gives an injured/suspended player. */
export function TransferListedBadge({ size = 18 }: { size?: number }) {
  const { t } = useTranslation()
  const label = t('playerPage.status.listedForTransferShort')

  return (
    <Tooltip title={label}>
      <Box
        role="img"
        aria-label={label}
        sx={{
          width: size,
          height: size,
          borderRadius: '50%',
          bgcolor: '#fff',
          border: '1px solid rgba(0,0,0,0.25)',
          display: 'inline-flex',
          alignItems: 'center',
          justifyContent: 'center',
          flexShrink: 0,
        }}
      >
        <SellIcon sx={{ fontSize: size * 0.65, color: 'success.main' }} />
      </Box>
    </Tooltip>
  )
}
