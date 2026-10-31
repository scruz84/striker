import Alert from '@mui/material/Alert'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { PageHeader } from '../../components/PageHeader'
import { LEGAL_EFFECTIVE_DATE, LEGAL_VERSION } from './legalMeta'
import { LegalLinks } from './LegalLinks'
import { useLegalValues } from './useLegalValues'

interface LegalSection {
  title: string
  paragraphs: string[]
}

type Translate = (key: string, options: Record<string, unknown>) => unknown

/** Shared body of the Privacy Policy and Terms pages: title, version and date, a warning when the
 * deployment's required legal details are missing, and the sections from the locale files with
 * the controller's identity filled in. `kind` selects legal.privacy.* or legal.terms.*. */
export function LegalDocument({ kind }: { kind: 'privacy' | 'terms' }) {
  const { t, i18n } = useTranslation()
  const { config, values, missingRequired } = useLegalValues()

  const sections = ((t as unknown as Translate)(`legal.${kind}.sections`, { returnObjects: true, ...values }) as LegalSection[]).map(
    (section) => ({ ...section, paragraphs: [...section.paragraphs] }),
  )
  if (kind === 'privacy') {
    // Optional facts are sentences of their own, dropped when the property is not set.
    if (config.postalAddress) {
      sections[0].paragraphs.push(t('legal.privacy.addressSentence', { address: config.postalAddress }))
    }
    if (config.hostingProvider) {
      const region = config.hostingRegion ? t('legal.privacy.hostingRegion', { region: config.hostingRegion }) : ''
      sections[3].paragraphs.push(t('legal.privacy.hostingSentence', { provider: config.hostingProvider, region }))
    }
  }

  const date = new Date(`${LEGAL_EFFECTIVE_DATE}T00:00:00Z`).toLocaleDateString(i18n.language, { dateStyle: 'long', timeZone: 'UTC' })

  return (
    <>
      <PageHeader title={t(`legal.${kind}.title`)} subtitle={t('legal.version', { version: LEGAL_VERSION, date })} />
      {missingRequired.length > 0 && (
        <Alert severity="warning" sx={{ mb: 2 }}>
          <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
            {t('legal.notConfigured')}
          </Typography>
          {t('legal.notConfiguredHelp', { missing: missingRequired.join(', ') })}
        </Alert>
      )}
      <Paper variant="outlined" sx={{ p: { xs: 2, sm: 3 } }}>
        <Stack spacing={3}>
          {sections.map((section) => (
            <section key={section.title}>
              <Typography variant="h6" component="h2" sx={{ mb: 1 }}>
                {section.title}
              </Typography>
              <Stack spacing={1}>
                {section.paragraphs.map((paragraph, index) => (
                  <Typography key={index} variant="body2" sx={{ overflowWrap: 'anywhere' }}>
                    {paragraph}
                  </Typography>
                ))}
              </Stack>
            </section>
          ))}
        </Stack>
      </Paper>
      <LegalLinks sx={{ mt: 2 }} />
    </>
  )
}
