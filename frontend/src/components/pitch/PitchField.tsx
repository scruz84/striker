import Box from '@mui/material/Box'
import { forwardRef, type ReactNode } from 'react'
import { PITCH_HEIGHT, PITCH_WIDTH } from './pitchLayout'

/** The striped-grass backdrop and line markings shared by every pitch view in the app (the
 * interactive lineup editor and the read-only match-details lineups) - children are absolutely
 * positioned slot contents laid out in the same 0-300 x 0-460 coordinate space as pitchLayout. */
export function PitchField({ children }: { children: ReactNode }) {
  return (
    <Box
      sx={{
        position: 'relative',
        width: '100%',
        aspectRatio: `${PITCH_WIDTH} / ${PITCH_HEIGHT}`,
        borderRadius: 2,
        overflow: 'hidden',
        background: 'repeating-linear-gradient(180deg, #3f8f46 0px, #3f8f46 40px, #388240 40px, #388240 80px)',
      }}
    >
      <PitchMarkings />
      {children}
    </Box>
  )
}

function PitchMarkings() {
  const line = 'rgba(255,255,255,0.7)'
  return (
    <svg
      viewBox={`0 0 ${PITCH_WIDTH} ${PITCH_HEIGHT}`}
      preserveAspectRatio="none"
      style={{ position: 'absolute', inset: 0, width: '100%', height: '100%', pointerEvents: 'none' }}
    >
      <rect x={4} y={4} width={PITCH_WIDTH - 8} height={PITCH_HEIGHT - 8} fill="none" stroke={line} strokeWidth={2} />
      <line x1={4} y1={PITCH_HEIGHT / 2} x2={PITCH_WIDTH - 4} y2={PITCH_HEIGHT / 2} stroke={line} strokeWidth={2} />
      <circle cx={PITCH_WIDTH / 2} cy={PITCH_HEIGHT / 2} r={42} fill="none" stroke={line} strokeWidth={2} />
      <circle cx={PITCH_WIDTH / 2} cy={PITCH_HEIGHT / 2} r={2.5} fill={line} />
      {/* own penalty + goal area, near the goalkeeper */}
      <rect x={70} y={PITCH_HEIGHT - 90} width={160} height={86} fill="none" stroke={line} strokeWidth={2} />
      <rect x={112} y={PITCH_HEIGHT - 34} width={76} height={30} fill="none" stroke={line} strokeWidth={2} />
      {/* opponent penalty + goal area, near the strikers */}
      <rect x={70} y={4} width={160} height={86} fill="none" stroke={line} strokeWidth={2} />
      <rect x={112} y={4} width={76} height={30} fill="none" stroke={line} strokeWidth={2} />
    </svg>
  )
}

/** Positions a child (a player chip, a placeholder) at a pitchLayout slot coordinate. Forwards
 * its ref to the positioned element itself - a droppable slot needs its hit-area ref on exactly
 * this element, not a wrapper around it. */
export const PitchSlotPosition = forwardRef<HTMLDivElement, { x: number; y: number; children: ReactNode }>(
  function PitchSlotPosition({ x, y, children }, ref) {
    return (
      <Box
        ref={ref}
        sx={{
          position: 'absolute',
          left: `${(x / PITCH_WIDTH) * 100}%`,
          top: `${(y / PITCH_HEIGHT) * 100}%`,
          transform: 'translate(-50%, -22px)',
        }}
      >
        {children}
      </Box>
    )
  },
)
