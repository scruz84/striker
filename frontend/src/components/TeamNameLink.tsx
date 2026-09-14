import Link, { type LinkProps } from '@mui/material/Link'
import { Link as RouterLink } from 'react-router-dom'

// A team's name, wherever it's shown standalone (not baked into a translated sentence), should
// link to that team's details page (issue #39) - this is the one place that wiring lives so every
// call site gets the same underline/color treatment instead of re-deriving it.
export function TeamNameLink({
  teamId,
  children,
  ...linkProps
}: { teamId: number; children: React.ReactNode } & Omit<LinkProps, 'component' | 'to'>) {
  return (
    <Link component={RouterLink} to={`/team/${teamId}`} underline="hover" color="inherit" {...linkProps}>
      {children}
    </Link>
  )
}
