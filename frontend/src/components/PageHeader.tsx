import type { ReactNode } from 'react'
import Box from '@mui/material/Box'
import Link from '@mui/material/Link'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

interface PageHeaderProps {
  title: ReactNode
  subtitle?: ReactNode
  /** Right-aligned control (a button, an icon button…). */
  action?: ReactNode
  /** Plain-string title for the browser tooltip when the visible one is ellipsised. */
  titleText?: string
}

/** The title block every page starts with: one line (ellipsised) on a phone, growing with the
 * viewport, with an optional subtitle and a right-aligned action. The page's context (club,
 * opponent…) belongs in the subtitle rather than in a long title. */
export function PageHeader({ title, subtitle, action, titleText }: PageHeaderProps) {
  return (
    <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mb: { xs: 1.5, sm: 2 }, minWidth: 0 }}>
      <Box sx={{ flexGrow: 1, minWidth: 0 }}>
        <Typography
          variant="pageTitle"
          component="h1"
          noWrap
          title={titleText}
          sx={{ fontSize: { xs: '1.375rem', sm: '1.625rem', md: '1.875rem' }, lineHeight: 1.25 }}
        >
          {title}
        </Typography>
        {subtitle && (
          <Typography variant="body2" color="text.secondary" component="div">
            {subtitle}
          </Typography>
        )}
      </Box>
      {action && <Box sx={{ flexShrink: 0 }}>{action}</Box>}
    </Stack>
  )
}

interface SectionHeaderProps {
  title: ReactNode
  /** Short "see all" link aligned to the right of the title. */
  seeAllTo?: string
  seeAllLabel?: string
  action?: ReactNode
}

/** Section title inside a page, with an optional short "Ver todo" link instead of a long
 * "Ver todos los partidos" that would wrap on a phone. */
export function SectionHeader({ title, seeAllTo, seeAllLabel, action }: SectionHeaderProps) {
  const { t } = useTranslation()
  return (
    <Stack direction="row" spacing={1} sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 1, minWidth: 0 }}>
      <Typography variant="subtitle1" component="h2" sx={{ fontWeight: 600, minWidth: 0 }} noWrap>
        {title}
      </Typography>
      {seeAllTo && (
        <Link
          component={RouterLink}
          to={seeAllTo}
          underline="hover"
          variant="body2"
          sx={{ flexShrink: 0, display: 'inline-flex', alignItems: 'center', minHeight: 44, px: 0.5 }}
        >
          {seeAllLabel ?? t('common.seeAll')}
        </Link>
      )}
      {action}
    </Stack>
  )
}
