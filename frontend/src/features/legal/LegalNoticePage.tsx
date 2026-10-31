import Alert from '@mui/material/Alert'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { PageHeader } from '../../components/PageHeader'
import { LegalLinks } from './LegalLinks'
import { useLegalValues } from './useLegalValues'

/** The "Aviso legal" (LSSI-CE identification of the service provider), only routed when the
 * deployment sets LEGAL_NOTICE_ENABLED=true - see App.tsx. Unset optional values are left out. */
export function LegalNoticePage() {
  const { t } = useTranslation()
  const { config, values, missingRequired } = useLegalValues()
  const rows = [
    { label: t('legal.notice.name'), value: values.controllerName },
    ...(config.taxId ? [{ label: t('legal.notice.taxId'), value: config.taxId }] : []),
    ...(config.postalAddress ? [{ label: t('legal.notice.address'), value: config.postalAddress }] : []),
    { label: t('legal.notice.email'), value: values.contactEmail },
  ]
  return (
    <>
      <PageHeader title={t('legal.notice.title')} subtitle={t('legal.notice.intro')} />
      {missingRequired.length > 0 && (
        <Alert severity="warning" sx={{ mb: 2 }}>
          <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
            {t('legal.notConfigured')}
          </Typography>
          {t('legal.notConfiguredHelp', { missing: missingRequired.join(', ') })}
        </Alert>
      )}
      <Paper variant="outlined" sx={{ p: { xs: 2, sm: 3 } }}>
        <Stack spacing={1.5} component="dl" sx={{ m: 0 }}>
          {rows.map((row) => (
            <div key={row.label}>
              <Typography variant="caption" color="text.secondary" component="dt">
                {row.label}
              </Typography>
              <Typography variant="body1" component="dd" sx={{ m: 0, overflowWrap: 'anywhere' }}>
                {row.value}
              </Typography>
            </div>
          ))}
        </Stack>
      </Paper>
      <LegalLinks sx={{ mt: 2 }} />
    </>
  )
}
