import { chance, createRng, pick } from './random'

export type HairStyle = 'bald' | 'short' | 'side' | 'curly' | 'mohawk'
export type BrowAngle = 'flat' | 'leftRaised' | 'rightRaised'
export type MouthStyle = 'flat' | 'smallSmile' | 'frown' | 'open'
export type FacialHair = 'none' | 'mustache' | 'goatee' | 'stubble'

export interface FaceTraits {
  skin: string
  hairStyle: HairStyle
  hairColor: string
  browAngle: BrowAngle
  mouthStyle: MouthStyle
  facialHair: FacialHair
}

// A blunt-outline "classic manager sim" palette - flat skin and hair tones, no background ring:
// the face is meant to sit directly on whatever surface it's placed on, the way this genre's
// player portraits traditionally do.
const SKIN_TONES = ['#F0C49A', '#E3AD7C', '#C98D5C', '#A66F42', '#7A4F30', '#F7DAB6'] as const
const HAIR_COLORS = ['#2B2320', '#5C3A21', '#8C5A1E', '#7A2E22', '#AFAAA0', '#3A2A1C'] as const

const HAIR_STYLES: readonly HairStyle[] = ['bald', 'short', 'side', 'curly', 'mohawk']
const BROW_ANGLES: readonly BrowAngle[] = ['flat', 'leftRaised', 'rightRaised']
const MOUTH_STYLES: readonly MouthStyle[] = ['flat', 'smallSmile', 'frown', 'open']

// derivePlayerFace is pure: the same seed (a player id works well) always yields the same
// face, so no backend storage is needed and the look survives re-renders and reloads.
export function derivePlayerFace(seed: number | string): FaceTraits {
  const rng = createRng(seed)

  let facialHair: FacialHair = 'none'
  if (chance(rng, 0.38)) {
    facialHair = pick(rng, ['mustache', 'goatee', 'stubble'] as const)
  }

  return {
    skin: pick(rng, SKIN_TONES),
    hairStyle: pick(rng, HAIR_STYLES),
    hairColor: pick(rng, HAIR_COLORS),
    browAngle: pick(rng, BROW_ANGLES),
    mouthStyle: pick(rng, MOUTH_STYLES),
    facialHair,
  }
}
