import { useState } from 'react'
import Button from '@mui/material/Button'
import Checkbox from '@mui/material/Checkbox'
import Container from '@mui/material/Container'
import FormControlLabel from '@mui/material/FormControlLabel'
import Link from '@mui/material/Link'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { Link as RouterLink, Navigate, useNavigate } from 'react-router-dom'
import { Trans, useTranslation } from 'react-i18next'
import { useAcceptTerms, useMe } from '../../api/hooks'
import { DeleteAccountDialog } from '../account/DeleteAccountDialog'
import { MINIMUM_AGE } from './legalMeta'
import { LegalLinks } from './LegalLinks'

/** The gate Layout sends every signed-in user to until they accepted the current Privacy Policy
 * and Terms (a first sign-in, or after the version was raised): confirm the minimum age and accept,
 * or delete the account instead. */
export function AcceptTermsPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { data: loggedUser } = useMe()
  const acceptTerms = useAcceptTerms()
  const [checked, setChecked] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  if (loggedUser === null) return <Navigate to="/login" replace />
  if (!loggedUser) return null
  if (loggedUser.termsAccepted) return <Navigate to="/dashboard" replace />

  const handleAccept = () => {
    acceptTerms.mutate({ version: loggedUser.termsVersion, ageConfirmed: true }, { onSuccess: () => navigate('/dashboard', { replace: true }) })
  }

  return (
    <Container maxWidth="sm" sx={{ py: { xs: 4, sm: 8 } }}>
      <Paper variant="outlined" sx={{ p: { xs: 2.5, sm: 4 } }}>
        <Stack spacing={2}>
          <Typography variant="h6" component="h1" sx={{ fontWeight: 600 }}>
            {t('acceptTerms.title')}
          </Typography>
          <Typography variant="body2">{t('acceptTerms.intro', { version: loggedUser.termsVersion })}</Typography>
          <Typography variant="body2">
            <Trans
              i18nKey="acceptTerms.read"
              components={{
                termsLink: <Link component={RouterLink} to="/terms" />,
                privacyLink: <Link component={RouterLink} to="/privacy" />,
              }}
              values={{ terms: t('legal.termsLink'), privacy: t('legal.privacyLink') }}
            />
          </Typography>
          <FormControlLabel
            control={<Checkbox checked={checked} onChange={(e) => setChecked(e.target.checked)} />}
            label={t('acceptTerms.checkbox', { age: MINIMUM_AGE })}
          />
          {acceptTerms.isError && (
            <Typography variant="body2" color="error">
              {t('acceptTerms.error')}
            </Typography>
          )}
          <Button variant="contained" disabled={!checked || acceptTerms.isPending} onClick={handleAccept}>
            {t('acceptTerms.accept')}
          </Button>
          <Typography variant="body2" color="text.secondary">
            {t('acceptTerms.orDelete')}
          </Typography>
          <Button color="error" variant="text" onClick={() => setDeleteOpen(true)} sx={{ alignSelf: 'flex-start' }}>
            {t('acceptTerms.deleteInstead')}
          </Button>
          <LegalLinks />
        </Stack>
      </Paper>
      <DeleteAccountDialog open={deleteOpen} onClose={() => setDeleteOpen(false)} teamName={loggedUser.teamName} />
    </Container>
  )
}
