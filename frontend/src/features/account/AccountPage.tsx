import { useState } from 'react'
import Alert from '@mui/material/Alert'
import Button from '@mui/material/Button'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { downloadMyData, useAccount, useMe } from '../../api/hooks'
import { PageHeader } from '../../components/PageHeader'
import { LegalLinks } from '../legal/LegalLinks'
import { DeleteAccountDialog } from './DeleteAccountDialog'

// Days before the club is released at which the user is warned (backend common.InactivityWarningDays
// is {46, 53}; shown as "days left").
const WARNING_DAYS_BEFORE_RELEASE = [14, 7]

/** The user's own account (/account): profile, the inactivity rules, "Download my data" (right of
 * access/portability) and "Delete my account" (right to erasure). */
export function AccountPage() {
  const { t, i18n } = useTranslation()
  const { data: loggedUser } = useMe()
  const { data: account, isLoading } = useAccount(!!loggedUser)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [downloadFailed, setDownloadFailed] = useState(false)

  if (loggedUser === null) return <Navigate to="/login" replace />
  if (!loggedUser || isLoading || !account) return <Skeleton variant="rectangular" height={240} />

  const formatDay = (iso: string) => new Date(iso).toLocaleDateString(i18n.language, { dateStyle: 'medium' })
  const rows: { label: string; value: string }[] = [
    { label: t('account.name'), value: account.name },
    { label: t('account.email'), value: account.email },
    { label: t('account.signInMethod'), value: account.provider },
    { label: t('account.country'), value: account.country || '-' },
    { label: t('account.club'), value: account.teamName || t('account.noClub') },
    { label: t('account.memberSince'), value: formatDay(account.createdAt) },
    { label: t('account.lastActive'), value: formatDay(account.lastSeenAt) },
    ...(account.termsVersion !== null && account.termsAcceptedAt
      ? [{ label: t('account.terms'), value: t('account.termsValue', { version: account.termsVersion, date: formatDay(account.termsAcceptedAt) }) }]
      : []),
  ]

  const handleDownload = () => {
    setDownloadFailed(false)
    downloadMyData().catch(() => setDownloadFailed(true))
  }

  return (
    <>
      <PageHeader title={t('account.title')} />
      <Stack spacing={2}>
        <Paper variant="outlined" sx={{ p: { xs: 2, sm: 3 } }}>
          <Typography variant="h6" component="h2" sx={{ mb: 1.5 }}>
            {t('account.profile')}
          </Typography>
          <Stack spacing={1.25} component="dl" sx={{ m: 0 }}>
            {rows.map((row) => (
              <div key={row.label}>
                <Typography variant="caption" color="text.secondary" component="dt">
                  {row.label}
                </Typography>
                <Typography variant="body2" component="dd" sx={{ m: 0, overflowWrap: 'anywhere' }}>
                  {row.value}
                </Typography>
              </div>
            ))}
          </Stack>
        </Paper>

        <Paper variant="outlined" sx={{ p: { xs: 2, sm: 3 } }}>
          <Typography variant="h6" component="h2" sx={{ mb: 1 }}>
            {t('account.rulesTitle')}
          </Typography>
          <Stack spacing={1}>
            <Typography variant="body2">
              {t('account.rulesRelease', {
                days: account.releaseAfterDays,
                first: WARNING_DAYS_BEFORE_RELEASE[0],
                second: WARNING_DAYS_BEFORE_RELEASE[1],
              })}
            </Typography>
            <Typography variant="body2">{t('account.rulesErase', { days: account.erasureAfterDays })}</Typography>
          </Stack>
        </Paper>

        <Paper variant="outlined" sx={{ p: { xs: 2, sm: 3 } }}>
          <Typography variant="h6" component="h2" sx={{ mb: 1 }}>
            {t('account.dataTitle')}
          </Typography>
          <Typography variant="body2" sx={{ mb: 1.5 }}>
            {t('account.dataText')}
          </Typography>
          <Button variant="outlined" onClick={handleDownload}>
            {t('account.download')}
          </Button>
          {downloadFailed && (
            <Alert severity="error" sx={{ mt: 1.5 }}>
              {t('account.downloadError')}
            </Alert>
          )}
        </Paper>

        <Paper variant="outlined" sx={{ p: { xs: 2, sm: 3 }, borderColor: 'error.main' }}>
          <Typography variant="h6" component="h2" color="error" sx={{ mb: 1 }}>
            {t('account.deleteTitle')}
          </Typography>
          <Typography variant="body2" sx={{ mb: 1.5 }}>
            {t('account.deleteText')}
          </Typography>
          <Button color="error" variant="outlined" onClick={() => setDeleteOpen(true)}>
            {t('account.deleteButton')}
          </Button>
        </Paper>

        <LegalLinks />
      </Stack>
      <DeleteAccountDialog open={deleteOpen} onClose={() => setDeleteOpen(false)} teamName={account.teamName} />
    </>
  )
}
