import { useMemo } from 'react'
import { derivePlayerFace } from './traits'
import type { FaceTraits } from './traits'

interface CartoonAvatarProps {
  /** Anything stable and unique per player - the player id is the usual choice. */
  seed: number | string
  size?: number
  className?: string
}

const INK = '#22201C'
const CX = 50
const CY = 58
const HEAD_RX = 25
const HEAD_RY = 27
const STROKE = 3.4

function Brows({ angle }: { angle: FaceTraits['browAngle'] }) {
  const positions = [
    { cx: 39, side: 'left' as const },
    { cx: 61, side: 'right' as const },
  ]
  return (
    <>
      {positions.map(({ cx, side }) => {
        const transform =
          angle === 'leftRaised' && side === 'left'
            ? `rotate(-8 ${cx} 45.5)`
            : angle === 'rightRaised' && side === 'right'
              ? `rotate(8 ${cx} 45.5)`
              : undefined
        return <rect key={cx} x={cx - 8} y={44} width={16} height={3.4} rx={1} fill={INK} transform={transform} />
      })}
    </>
  )
}

// Eyes are always plain, open ovals - a player roster reads badly with anyone winking or
// squinting, so this style deliberately has no closed-eye variant.
function Eyes() {
  return (
    <>
      {[39, 61].map((cx) => (
        <g key={cx}>
          <ellipse cx={cx} cy={51} rx={5.4} ry={4.4} fill="#ffffff" stroke={INK} strokeWidth={2} />
          <circle cx={cx} cy={51.5} r={2.1} fill={INK} />
        </g>
      ))}
    </>
  )
}

// The one feature most cartoon-avatar generators skip - a simple side-profile hint is what
// makes this style read as a "classic manager sim" portrait rather than a generic blob face.
function Nose() {
  return (
    <path
      d="M 49 53 Q 46.5 61 48.5 65 L 53 65.5"
      stroke={INK}
      strokeWidth={2.2}
      fill="none"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
  )
}

function Mouth({ style }: { style: FaceTraits['mouthStyle'] }) {
  switch (style) {
    case 'smallSmile':
      return <path d="M 42 74 Q 50 79 58 74" stroke={INK} strokeWidth={2.6} strokeLinecap="round" fill="none" />
    case 'frown':
      return <path d="M 42 77 Q 50 73 58 77" stroke={INK} strokeWidth={2.6} strokeLinecap="round" fill="none" />
    case 'open':
      return <rect x={44} y={72} width={12} height={6} rx={2.5} fill={INK} />
    case 'flat':
    default:
      return <line x1={43} y1={75} x2={57} y2={75} stroke={INK} strokeWidth={2.6} strokeLinecap="round" />
  }
}

function FacialHair({ style }: { style: FaceTraits['facialHair'] }) {
  switch (style) {
    case 'mustache':
      return <rect x={40} y={67} width={20} height={4.2} rx={2} fill={INK} />
    case 'goatee':
      return <path d="M 43 76 Q 50 90 57 76 Q 50 82 43 76 Z" fill={INK} />
    case 'stubble':
      return (
        <>
          {Array.from({ length: 24 }, (_, i) => {
            const x = CX - 20 + (i % 12) * 3.6
            const y = 68 + Math.floor(i / 12) * 8
            return <circle key={i} cx={x} cy={y} r={0.7} fill={INK} opacity={0.5} />
          })}
        </>
      )
    default:
      return null
  }
}

function Hair({ style, color, skin }: { style: FaceTraits['hairStyle']; color: string; skin: string }) {
  switch (style) {
    case 'short':
      return (
        <path
          d="M 22 44 Q 21 17 50 16 Q 79 17 78 44 Q 74 26 50 25 Q 26 26 22 44 Z"
          fill={color}
          stroke={INK}
          strokeWidth={STROKE}
          strokeLinejoin="round"
        />
      )
    case 'side':
      return (
        <path
          d="M 21 42 Q 22 16 50 17 Q 42 29 73 25 L 82 42 L 82 13 L 21 13 Z"
          fill={color}
          stroke={INK}
          strokeWidth={STROKE}
          strokeLinejoin="round"
        />
      )
    case 'curly':
      return (
        <>
          {([
            [27, 33],
            [37, 23],
            [50, 18],
            [63, 23],
            [73, 33],
          ] as const).map(([bx, by], i) => (
            <circle key={i} cx={bx} cy={by} r={8.5} fill={color} stroke={INK} strokeWidth={2.6} />
          ))}
        </>
      )
    case 'mohawk':
      return (
        <>
          <path d="M 41 30 L 50 8 L 59 30 Q 50 22 41 30 Z" fill={color} stroke={INK} strokeWidth={2.8} strokeLinejoin="round" />
          <path
            d="M 22 42 Q 30 34 50 33 Q 70 34 78 42 Q 73 40 50 39 Q 27 40 22 42 Z"
            fill={skin}
            stroke={INK}
            strokeWidth={2.4}
          />
        </>
      )
    case 'cap':
      return (
        <>
          <path
            d="M 19 40 Q 20 18 50 18 Q 80 18 81 40 L 81 30 Q 50 20 19 30 Z"
            fill={color}
            stroke={INK}
            strokeWidth={STROKE}
            strokeLinejoin="round"
          />
          <rect x={19} y={26} width={62} height={6} fill={color} stroke={INK} strokeWidth={2.2} />
        </>
      )
    case 'bald':
    default:
      return null
  }
}

/** An original, deterministic "classic manager sim" cartoon face - blunt ink outlines, a
 * visible nose, no background ring, sitting directly on whatever surface it's placed on. See
 * traits.ts for how the look is derived. */
export function CartoonAvatar({ seed, size = 56, className }: CartoonAvatarProps) {
  const traits = useMemo(() => derivePlayerFace(seed), [seed])

  return (
    <svg viewBox="0 0 100 100" width={size} height={size} className={className} role="img" aria-hidden="true">
      <ellipse cx={20} cy={60} rx={6} ry={9} fill={traits.skin} stroke={INK} strokeWidth={2.6} />
      <ellipse cx={80} cy={60} rx={6} ry={9} fill={traits.skin} stroke={INK} strokeWidth={2.6} />
      <ellipse cx={CX} cy={CY} rx={HEAD_RX} ry={HEAD_RY} fill={traits.skin} stroke={INK} strokeWidth={STROKE} />

      {traits.facialHair === 'stubble' && <FacialHair style={traits.facialHair} />}

      <Brows angle={traits.browAngle} />
      <Eyes />
      <Nose />
      <Mouth style={traits.mouthStyle} />
      {traits.facialHair !== 'stubble' && <FacialHair style={traits.facialHair} />}

      <Hair style={traits.hairStyle} color={traits.hairColor} skin={traits.skin} />
    </svg>
  )
}
