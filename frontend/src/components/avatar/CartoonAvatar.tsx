import { useId, useMemo } from 'react'
import { derivePlayerFace } from './traits'
import type { FaceTraits } from './traits'

interface CartoonAvatarProps {
  /** Anything stable and unique per player - the player id is the usual choice. */
  seed: number | string
  size?: number
  className?: string
}

const CX = 50
const CY = 58
const HEAD_RX = 30
const HEAD_RY = 32

function darken(hex: string, amount: number): string {
  const n = parseInt(hex.slice(1), 16)
  const clamp = (v: number) => Math.max(0, Math.min(255, v))
  const r = clamp(((n >> 16) & 0xff) * (1 - amount))
  const g = clamp(((n >> 8) & 0xff) * (1 - amount))
  const b = clamp((n & 0xff) * (1 - amount))
  return `rgb(${r | 0}, ${g | 0}, ${b | 0})`
}

/** A point on the head's silhouette at angleDeg (0 = right, 90 = down, 270/-90 = top). */
function headPoint(angleDeg: number, rx = HEAD_RX, ry = HEAD_RY) {
  const a = (angleDeg * Math.PI) / 180
  return { x: CX + rx * Math.cos(a), y: CY + ry * Math.sin(a) }
}

function Eyebrows({ style, color }: { style: FaceTraits['eyebrowStyle']; color: string }) {
  const positions = [
    { cx: 38, mirror: 1 },
    { cx: 62, mirror: -1 },
  ]
  return (
    <>
      {positions.map(({ cx, mirror }) => {
        switch (style) {
          case 'raised':
            return (
              <path
                key={cx}
                d={`M ${cx - 8} 40 Q ${cx} 32 ${cx + 8} 38`}
                stroke={color}
                strokeWidth={3}
                strokeLinecap="round"
                fill="none"
              />
            )
          case 'angry':
            return (
              <rect
                key={cx}
                x={cx - 8}
                y={37}
                width={16}
                height={3.2}
                rx={1.6}
                fill={color}
                transform={`rotate(${12 * mirror} ${cx} 38.5)`}
              />
            )
          case 'thin':
            return <rect key={cx} x={cx - 6} y={39} width={12} height={2} rx={1} fill={color} />
          case 'flat':
          default:
            return <rect key={cx} x={cx - 8} y={38} width={16} height={3} rx={1.5} fill={color} />
        }
      })}
    </>
  )
}

function Eyes({ style }: { style: FaceTraits['eyeStyle'] }) {
  const eyeRx = style === 'wide' ? 10 : 8.5
  const eyeRy = style === 'wide' ? 13 : 11
  const positions = [
    { cx: 38, closed: false },
    { cx: 62, closed: style === 'wink' },
  ]
  return (
    <>
      {positions.map(({ cx, closed }) => {
        if (closed) {
          return (
            <path
              key={cx}
              d={`M ${cx - 8} 52 Q ${cx} 58 ${cx + 8} 52`}
              stroke="#2b2b2b"
              strokeWidth={2.5}
              strokeLinecap="round"
              fill="none"
            />
          )
        }
        const lidCover = style === 'sleepy' ? eyeRy * 0.55 : 0
        return (
          <g key={cx}>
            <ellipse cx={cx} cy={52} rx={eyeRx} ry={eyeRy} fill="#ffffff" />
            <circle cx={cx} cy={54} r={4.6} fill="#26282b" />
            {lidCover > 0 && (
              <rect x={cx - eyeRx - 1} y={52 - eyeRy - 1} width={2 * eyeRx + 2} height={lidCover + 1} fill="#2b2b2b" />
            )}
          </g>
        )
      })}
    </>
  )
}

function Mouth({ style }: { style: FaceTraits['mouthStyle'] }) {
  switch (style) {
    case 'smile':
      return <path d="M 40 76 Q 50 87 60 76" stroke="#7a3b2e" strokeWidth={3} strokeLinecap="round" fill="none" />
    case 'smirk':
      return <path d="M 42 78 Q 53 83 58 75" stroke="#7a3b2e" strokeWidth={3} strokeLinecap="round" fill="none" />
    case 'flat':
      return <line x1={41} y1={78} x2={59} y2={78} stroke="#7a3b2e" strokeWidth={3} strokeLinecap="round" />
    case 'open':
      return (
        <g>
          <ellipse cx={50} cy={80} rx={8} ry={6.5} fill="#6b2c2c" />
          <rect x={43} y={75} width={14} height={4} rx={2} fill="#ffffff" />
        </g>
      )
    case 'small':
    default:
      return <ellipse cx={50} cy={78} rx={3} ry={3.4} fill="#7a3b2e" />
  }
}

function FacialHair({ style, color }: { style: FaceTraits['facialHair']; color: string }) {
  switch (style) {
    case 'mustache':
      return (
        <g>
          <ellipse cx={43} cy={71} rx={6} ry={3} fill={color} />
          <ellipse cx={57} cy={71} rx={6} ry={3} fill={color} />
        </g>
      )
    case 'goatee':
      return <path d="M 43 80 Q 50 96 57 80 Q 50 88 43 80 Z" fill={color} />
    case 'beard':
      return <path d="M 21 52 Q 24 92 50 96 Q 76 92 79 52 Q 79 78 50 82 Q 21 78 21 52 Z" fill={color} />
    default:
      return null
  }
}

function Accessory({ style }: { style: FaceTraits['accessory'] }) {
  if (style === 'none') return null
  const lens = style === 'sunglasses' ? '#2b2b2b' : 'none'
  return (
    <g stroke="#2b2b2b" strokeWidth={2} fill={lens}>
      <circle cx={38} cy={52} r={11} />
      <circle cx={62} cy={52} r={11} />
      <line x1={49} y1={51} x2={51} y2={51} />
      <line x1={27} y1={50} x2={20} y2={47} />
      <line x1={73} y1={50} x2={80} y2={47} />
    </g>
  )
}

function Hair({ style, color, clipId }: { style: FaceTraits['hairStyle']; color: string; clipId: string }) {
  switch (style) {
    case 'bald':
      return null
    case 'buzz':
      return (
        <g clipPath={`url(#${clipId})`}>
          <rect x={20} y={26} width={60} height={14} fill={color} opacity={0.55} />
        </g>
      )
    case 'short':
      return (
        <g clipPath={`url(#${clipId})`}>
          <rect x={18} y={22} width={64} height={24} fill={color} />
        </g>
      )
    case 'side':
      return (
        <g clipPath={`url(#${clipId})`}>
          <path d="M 18 44 Q 20 20 50 22 Q 46 34 70 30 L 82 44 L 82 20 L 18 20 Z" fill={color} />
        </g>
      )
    case 'curly': {
      const bumps = [200, 225, 250, 270, 290, 315, 340].map((angle) => headPoint(angle, HEAD_RX - 2, HEAD_RY - 4))
      return (
        <g>
          {bumps.map((p, i) => (
            <circle key={i} cx={p.x} cy={p.y - 2} r={7.5} fill={color} />
          ))}
        </g>
      )
    }
    case 'mohawk': {
      const spine = [260, 270, 280].map((angle) => headPoint(angle, HEAD_RX - 10, HEAD_RY - 4))
      return (
        <g>
          {spine.map((p, i) => (
            <path
              key={i}
              d={`M ${p.x - 5} ${p.y + 6} L ${p.x} ${p.y - 12} L ${p.x + 5} ${p.y + 6} Z`}
              fill={color}
            />
          ))}
        </g>
      )
    }
    case 'spiky': {
      const spikes = [200, 220, 240, 260, 280, 300, 320, 340].map((angle) => headPoint(angle))
      return (
        <g>
          {spikes.map((p, i) => {
            const outward = headPoint([200, 220, 240, 260, 280, 300, 320, 340][i], HEAD_RX + 12, HEAD_RY + 12)
            return <path key={i} d={`M ${p.x - 4} ${p.y} L ${outward.x} ${outward.y} L ${p.x + 4} ${p.y} Z`} fill={color} />
          })}
        </g>
      )
    }
    default:
      return null
  }
}

/** An original, deterministic cartoon face - see traits.ts for how the look is derived. */
export function CartoonAvatar({ seed, size = 56, className }: CartoonAvatarProps) {
  const traits = useMemo(() => derivePlayerFace(seed), [seed])
  const clipId = useId()
  const skinShadow = darken(traits.skin, 0.15)

  return (
    <svg
      viewBox="0 0 100 100"
      width={size}
      height={size}
      className={className}
      role="img"
      aria-hidden="true"
    >
      <defs>
        <clipPath id={clipId}>
          <ellipse cx={CX} cy={CY} rx={HEAD_RX} ry={HEAD_RY} />
        </clipPath>
      </defs>

      <circle cx={50} cy={50} r={50} fill={traits.ringColor} />

      <circle cx={30} cy={58} r={7} fill={traits.skin} />
      <circle cx={70} cy={58} r={7} fill={traits.skin} />
      <ellipse cx={CX} cy={CY} rx={HEAD_RX} ry={HEAD_RY} fill={traits.skin} />

      {traits.facialHair === 'beard' && <FacialHair style={traits.facialHair} color={traits.hairColor} />}

      <Eyebrows style={traits.eyebrowStyle} color={skinShadow} />
      <Eyes style={traits.eyeStyle} />
      <path d="M 50 58 Q 47 64 50 67" stroke={skinShadow} strokeWidth={1.6} fill="none" opacity={0.5} />
      <Mouth style={traits.mouthStyle} />
      {traits.facialHair !== 'beard' && <FacialHair style={traits.facialHair} color={traits.hairColor} />}

      <Hair style={traits.hairStyle} color={traits.hairColor} clipId={clipId} />
      <Accessory style={traits.accessory} />
    </svg>
  )
}
