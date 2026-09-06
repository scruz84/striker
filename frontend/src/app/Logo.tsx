import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'

// Simplified soccer-ball glyph (circle + one pentagon panel + its seams) - the same shape as
// public/favicon.svg, redrawn in `currentColor` so it always matches the header's text color in
// both light and dark themes instead of carrying its own fixed palette.
function BallMark({ size = 26 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 64 64" fill="none" aria-hidden="true">
      <circle cx="32" cy="32" r="29" stroke="currentColor" strokeWidth="4" />
      <g stroke="currentColor" strokeWidth="4.5" strokeLinecap="round">
        <line x1="32" y1="18" x2="32" y2="9" />
        <line x1="45.31" y1="27.67" x2="53.87" y2="24.89" />
        <line x1="40.23" y1="43.33" x2="45.52" y2="50.61" />
        <line x1="23.77" y1="43.33" x2="18.48" y2="50.61" />
        <line x1="18.69" y1="27.67" x2="10.13" y2="24.89" />
      </g>
      <polygon points="32,18 45.31,27.67 40.23,43.33 23.77,43.33 18.69,27.67" fill="currentColor" />
    </svg>
  )
}

// The header wordmark - a condensed, bold display face (Oswald, self-hosted via @fontsource so
// it renders identically offline) paired with the ball glyph above, standing apart from the
// app's regular MUI/system body font on purpose, the way a masthead usually does.
export function Logo() {
  return (
    <Box sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.75 }}>
      <BallMark />
      <Typography
        component="span"
        sx={{
          fontFamily: '"Oswald", sans-serif',
          fontWeight: 700,
          fontSize: '1.6rem',
          letterSpacing: '0.05em',
          textTransform: 'uppercase',
          lineHeight: 1,
        }}
      >
        Striker
      </Typography>
    </Box>
  )
}
