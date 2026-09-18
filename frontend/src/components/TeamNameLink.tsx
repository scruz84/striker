import Link, { type LinkProps } from '@mui/material/Link'
import Typography, { type TypographyProps } from '@mui/material/Typography'
import { Link as RouterLink, useNavigate } from 'react-router-dom'

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

// A team's name nested inside a row/card that's itself a link or button to somewhere else (a
// match, a cup tie, a friendly) - a real <a> (what TeamNameLink renders) can't nest inside
// another <a> or <button>, so this renders as a span that navigates on click/Enter/Space instead,
// stopping the click from also triggering the parent row's own navigation.
export function NestedTeamNameLink({
  teamId,
  children,
  sx,
  ...typographyProps
}: { teamId: number; children: React.ReactNode } & Omit<TypographyProps, 'component' | 'onClick' | 'onKeyDown'>) {
  const navigate = useNavigate()
  const go = (event: { stopPropagation: () => void; preventDefault: () => void }) => {
    event.stopPropagation()
    event.preventDefault()
    navigate(`/team/${teamId}`)
  }
  return (
    <Typography
      component="span"
      role="link"
      tabIndex={0}
      onClick={go}
      onKeyDown={(event) => {
        if (event.key === 'Enter' || event.key === ' ') go(event)
      }}
      sx={{ cursor: 'pointer', '&:hover': { textDecoration: 'underline' }, ...sx }}
      {...typographyProps}
    >
      {children}
    </Typography>
  )
}
