import { chance, createRng, pick } from './random'

export type HairStyle = 'bald' | 'buzz' | 'short' | 'side' | 'spiky' | 'curly' | 'mohawk'
export type EyeStyle = 'round' | 'wide' | 'sleepy' | 'wink'
export type EyebrowStyle = 'flat' | 'raised' | 'angry' | 'thin'
export type MouthStyle = 'smile' | 'smirk' | 'flat' | 'open' | 'small'
export type FacialHair = 'none' | 'mustache' | 'beard' | 'goatee'
export type Accessory = 'none' | 'glasses' | 'sunglasses'

export interface FaceTraits {
  skin: string
  ringColor: string
  hairStyle: HairStyle
  hairColor: string
  eyeStyle: EyeStyle
  eyebrowStyle: EyebrowStyle
  mouthStyle: MouthStyle
  facialHair: FacialHair
  accessory: Accessory
}

// A curated cartoon palette - the warm yellow is a nod to classic animated-sitcom character
// design, mixed with ordinary skin tones so it reads as "one option among many", not a
// mandatory reproduction of any specific show's cast.
const SKIN_TONES = ['#FFD83D', '#F1C27D', '#E0AC69', '#C68642', '#8D5524', '#FFDBAC'] as const
const RING_COLORS = ['#4CAF50', '#2196F3', '#FF9800', '#E91E63', '#9C27B0', '#00BCD4', '#795548'] as const
const HAIR_COLORS = ['#2C2C2C', '#6B4226', '#D4A017', '#B33F27', '#B0B0B0', '#3B2B20'] as const

const HAIR_STYLES: readonly HairStyle[] = ['bald', 'buzz', 'short', 'side', 'spiky', 'curly', 'mohawk']
const EYE_STYLES: readonly EyeStyle[] = ['round', 'wide', 'sleepy', 'wink']
const EYEBROW_STYLES: readonly EyebrowStyle[] = ['flat', 'raised', 'angry', 'thin']
const MOUTH_STYLES: readonly MouthStyle[] = ['smile', 'smirk', 'flat', 'open', 'small']

// derivePlayerFace is pure: the same seed (a player id works well) always yields the same
// face, so no backend storage is needed and the look survives re-renders and reloads.
export function derivePlayerFace(seed: number | string): FaceTraits {
  const rng = createRng(seed)

  let facialHair: FacialHair = 'none'
  if (chance(rng, 0.35)) {
    facialHair = pick(rng, ['mustache', 'beard', 'goatee'] as const)
  }

  let accessory: Accessory = 'none'
  if (chance(rng, 0.2)) {
    accessory = pick(rng, ['glasses', 'sunglasses'] as const)
  }

  return {
    skin: pick(rng, SKIN_TONES),
    ringColor: pick(rng, RING_COLORS),
    hairStyle: pick(rng, HAIR_STYLES),
    hairColor: pick(rng, HAIR_COLORS),
    eyeStyle: pick(rng, EYE_STYLES),
    eyebrowStyle: pick(rng, EYEBROW_STYLES),
    mouthStyle: pick(rng, MOUTH_STYLES),
    facialHair,
    accessory,
  }
}
