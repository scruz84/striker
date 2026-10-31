import { useState } from 'react'
import Button from '@mui/material/Button'
import Dialog from '@mui/material/Dialog'
import DialogActions from '@mui/material/DialogActions'
import DialogContent from '@mui/material/DialogContent'
import DialogContentText from '@mui/material/DialogContentText'
import DialogTitle from '@mui/material/DialogTitle'
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useDeleteAccount } from '../../api/hooks'
import { isDeleteConfirmation } from './deleteConfirmation'

/** Confirms and performs the self-service account deletion. The delete button only enables once
 * the user typed their club's exact name (or DELETE/BORRAR when they have no club). On success the
 * session is gone, so it sends them to the home page. */
export function DeleteAccountDialog({ open, onClose, teamName }: { open: boolean; onClose: () => void; teamName: string }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const deleteAccount = useDeleteAccount()
  const [typed, setTyped] = useState('')

  const handleClose = () => {
    setTyped('')
    onClose()
  }
  const handleDelete = () => {
    deleteAccount.mutate(typed.trim(), { onSuccess: () => navigate('/', { replace: true }) })
  }

  return (
    <Dialog open={open} onClose={handleClose} fullWidth maxWidth="xs">
      <DialogTitle>{t('deleteAccount.title')}</DialogTitle>
      <DialogContent>
        <DialogContentText sx={{ mb: 2 }}>{t('deleteAccount.body')}</DialogContentText>
        <Typography variant="body2" sx={{ mb: 1, overflowWrap: 'anywhere' }}>
          {teamName ? t('deleteAccount.confirmClub', { team: teamName }) : t('deleteAccount.confirmNoClub')}
        </Typography>
        <TextField
          value={typed}
          onChange={(e) => setTyped(e.target.value)}
          label={t('deleteAccount.inputLabel')}
          fullWidth
          size="small"
          autoComplete="off"
        />
        {deleteAccount.isError && (
          <Typography variant="body2" color="error" sx={{ mt: 1 }}>
            {t('deleteAccount.error')}
          </Typography>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose}>{t('deleteAccount.cancel')}</Button>
        <Button color="error" variant="contained" disabled={!isDeleteConfirmation(typed, teamName) || deleteAccount.isPending} onClick={handleDelete}>
          {t('deleteAccount.confirm')}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
