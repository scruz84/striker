import Link, { type LinkProps } from '@mui/material/Link'
import { Link as RouterLink } from 'react-router-dom'

// A player's name, wherever it's shown standalone (not baked into a translated sentence, e.g.
// match-event commentary), should link to that player's details page (issue #41) - this is the
// one place that wiring lives so every call site gets the same underline/color treatment instead
// of re-deriving it. Mirrors TeamNameLink.
export function PlayerNameLink({
  playerId,
  children,
  ...linkProps
}: { playerId: number; children: React.ReactNode } & Omit<LinkProps, 'component' | 'to'>) {
  return (
    <Link component={RouterLink} to={`/player/${playerId}`} underline="hover" color="inherit" {...linkProps}>
      {children}
    </Link>
  )
}
