import Link from '@mui/material/Link'
import Stack from '@mui/material/Stack'
import type { SxProps, Theme } from '@mui/material/styles'
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { getLegalConfig } from '../../config'

/** Links to the Privacy Policy and Terms of Use (and the legal notice when the deployment enabled
 * it with LEGAL_NOTICE_ENABLED). Shown on the login page, help page, navigation drawer, account
 * page and the legal pages themselves. */
export function LegalLinks({ sx, onNavigate }: { sx?: SxProps<Theme>; onNavigate?: () => void }) {
  const { t } = useTranslation()
  const noticeEnabled = getLegalConfig().noticeEnabled
  const links = [
    { to: '/privacy', label: t('legal.privacyLink') },
    { to: '/terms', label: t('legal.termsLink') },
    ...(noticeEnabled ? [{ to: '/legal-notice', label: t('legal.noticeLink') }] : []),
  ]
  return (
    <Stack direction="row" useFlexGap sx={[{ flexWrap: 'wrap', columnGap: 2, rowGap: 0.5 }, ...(Array.isArray(sx) ? sx : [sx])]}>
      {links.map((link) => (
        <Link key={link.to} component={RouterLink} to={link.to} variant="body2" underline="hover" onClick={onNavigate}>
          {link.label}
        </Link>
      ))}
    </Stack>
  )
}
