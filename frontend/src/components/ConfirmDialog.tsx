import { useState, type ReactNode } from 'react'
import Button from '@mui/material/Button'
import Dialog from '@mui/material/Dialog'
import DialogActions from '@mui/material/DialogActions'
import DialogContent from '@mui/material/DialogContent'
import DialogContentText from '@mui/material/DialogContentText'
import DialogTitle from '@mui/material/DialogTitle'
import { useTranslation } from 'react-i18next'
import { ConfirmContext, type ConfirmFn, type ConfirmOptions } from './confirmDialogContext'

interface PendingConfirm extends ConfirmOptions {
  resolve: (result: boolean) => void
}

export function ConfirmProvider({ children }: { children: ReactNode }) {
  const { t } = useTranslation()
  const [pending, setPending] = useState<PendingConfirm | null>(null)

  const confirm: ConfirmFn = (options) => new Promise<boolean>((resolve) => setPending({ ...options, resolve }))

  const settle = (result: boolean) => {
    pending?.resolve(result)
    setPending(null)
  }

  return (
    <ConfirmContext.Provider value={confirm}>
      {children}
      <Dialog open={!!pending} onClose={() => settle(false)} maxWidth="xs" fullWidth slotProps={{ paper: { sx: { m: 2, width: 'calc(100% - 32px)' } } }}>
        <DialogTitle sx={{ overflowWrap: 'anywhere' }}>{pending?.title ?? t('common.confirmTitle')}</DialogTitle>
        <DialogContent>
          <DialogContentText component="div">{pending?.message}</DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => settle(false)}>{t('common.cancel')}</Button>
          <Button
            onClick={() => settle(true)}
            color={pending?.destructive ? 'error' : 'primary'}
            variant="contained"
            autoFocus
          >
            {pending?.confirmLabel ?? t('common.confirm')}
          </Button>
        </DialogActions>
      </Dialog>
    </ConfirmContext.Provider>
  )
}
