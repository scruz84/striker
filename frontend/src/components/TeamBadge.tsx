import Box from '@mui/material/Box'
import { useMemo } from 'react'
import { teamColor } from '../utils/teamColors'

function initialsFor(teamName: string): string {
  const words = teamName.trim().split(/\s+/).filter(Boolean)
  if (words.length === 0) return '?'
  if (words.length === 1) return words[0].slice(0, 2).toUpperCase()
  return (words[0][0] + words[words.length - 1][0]).toUpperCase()
}

/** A colored "shield" of a team's initials, deterministic per team id - teams have no crest
 * artwork, so this stands in as a lightweight visual identity wherever a team needs one. */
export function TeamBadge({ teamId, teamName, size = 40 }: { teamId: number; teamName: string; size?: number }) {
  const color = useMemo(() => teamColor(teamId), [teamId])
  const initials = initialsFor(teamName)

  return (
    <Box
      sx={{
        width: size,
        height: size,
        flexShrink: 0,
        bgcolor: color,
        color: '#fff',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontWeight: 700,
        fontSize: size * 0.4,
        borderRadius: '18% 18% 50% 50%',
      }}
    >
      {initials}
    </Box>
  )
}
