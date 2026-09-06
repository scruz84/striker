import Box from '@mui/material/Box'
import { forwardRef, type ReactNode } from 'react'
import { PITCH_HEIGHT, PITCH_HEIGHT_HALF, PITCH_WIDTH } from './pitchLayout'

/** The striped-grass backdrop and line markings shared by every pitch view in the app. Children
 * are absolutely positioned slot contents laid out in the same coordinate space as
 * pitchLayout.layoutPitchSlots - pass the same `half` value to both. `half` shows only the
 * occupied own-half-plus-halfway-line (PITCH_HEIGHT_HALF tall) instead of the full pitch - see
 * PITCH_HEIGHT_HALF's comment in pitchLayout.ts. */
export function PitchField({ children, half = false }: { children: ReactNode; half?: boolean }) {
  const height = half ? PITCH_HEIGHT_HALF : PITCH_HEIGHT
  return (
    <Box
      sx={{
        position: 'relative',
        width: '100%',
        aspectRatio: `${PITCH_WIDTH} / ${height}`,
        borderRadius: 2,
        overflow: 'hidden',
        background: 'repeating-linear-gradient(180deg, #3f8f46 0px, #3f8f46 40px, #388240 40px, #388240 80px)',
      }}
    >
      <PitchMarkings half={half} />
      {children}
    </Box>
  )
}

function PitchMarkings({ half }: { half: boolean }) {
  const line = 'rgba(255,255,255,0.7)'
  const height = half ? PITCH_HEIGHT_HALF : PITCH_HEIGHT
  const centerX = PITCH_WIDTH / 2

  return (
    <svg
      viewBox={`0 0 ${PITCH_WIDTH} ${height}`}
      preserveAspectRatio="none"
      style={{ position: 'absolute', inset: 0, width: '100%', height: '100%', pointerEvents: 'none' }}
    >
      <rect x={4} y={4} width={PITCH_WIDTH - 8} height={height - 8} fill="none" stroke={line} strokeWidth={2} />
      {/* own penalty + goal area, near the goalkeeper */}
      <rect x={70} y={height - 90} width={160} height={86} fill="none" stroke={line} strokeWidth={2} />
      <rect x={112} y={height - 34} width={76} height={30} fill="none" stroke={line} strokeWidth={2} />
      {half ? (
        // Halfway line along the top edge, with the center circle cut off by it (only the arc
        // bulging into the visible half is drawn) - the standard "half pitch" broadcast graphic.
        <path
          d={`M ${centerX - 42} 4 A 42 42 0 0 0 ${centerX + 42} 4`}
          fill="none"
          stroke={line}
          strokeWidth={2}
        />
      ) : (
        <>
          <line x1={4} y1={height / 2} x2={PITCH_WIDTH - 4} y2={height / 2} stroke={line} strokeWidth={2} />
          <circle cx={centerX} cy={height / 2} r={42} fill="none" stroke={line} strokeWidth={2} />
          <circle cx={centerX} cy={height / 2} r={2.5} fill={line} />
          {/* opponent penalty + goal area, near the strikers */}
          <rect x={70} y={4} width={160} height={86} fill="none" stroke={line} strokeWidth={2} />
          <rect x={112} y={4} width={76} height={30} fill="none" stroke={line} strokeWidth={2} />
        </>
      )}
    </svg>
  )
}

/** Positions a child (a player chip, a placeholder) at a pitchLayout slot coordinate. Forwards
 * its ref to the positioned element itself - a droppable slot needs its hit-area ref on exactly
 * this element, not a wrapper around it. Pass the same `pitchHeight` the slot's y was computed
 * against (PITCH_HEIGHT_HALF for a `half` PitchField, PITCH_HEIGHT otherwise). */
export const PitchSlotPosition = forwardRef<
  HTMLDivElement,
  { x: number; y: number; pitchHeight?: number; children: ReactNode }
>(function PitchSlotPosition({ x, y, pitchHeight = PITCH_HEIGHT, children }, ref) {
  return (
    <Box
      ref={ref}
      sx={{
        position: 'absolute',
        left: `${(x / PITCH_WIDTH) * 100}%`,
        top: `${(y / pitchHeight) * 100}%`,
        transform: 'translate(-50%, -22px)',
      }}
    >
      {children}
    </Box>
  )
})
