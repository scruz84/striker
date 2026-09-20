import { chance, createRng, pick, pickWeighted } from './random'

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

// A player's face should skew toward what's common for their birth region while still leaving
// every skin tone reachable (issue #71) - nobody's excluded, but the odds aren't uniform either.
export type Region = 'europe' | 'africa' | 'asia' | 'americas'

// Mirrors backend common.KnownCountries - every country code the game can currently generate a
// world for. An unrecognized/missing code (legacy players created before birthCountry existed)
// falls back to 'europe'.
const COUNTRY_REGIONS: Record<string, Region> = {
  es: 'europe',
  en: 'europe',
  fr: 'europe',
  de: 'europe',
  it: 'europe',
  pt: 'europe',
  nl: 'europe',
  se: 'europe',
  pl: 'europe',
  gr: 'europe',
  br: 'americas',
  ar: 'americas',
  mx: 'americas',
  co: 'americas',
  cl: 'americas',
  pe: 'americas',
  uy: 'americas',
  us: 'americas',
  ca: 'americas',
  ec: 'americas',
  jp: 'asia',
  kr: 'asia',
  cn: 'asia',
  in: 'asia',
  th: 'asia',
  vn: 'asia',
  id: 'asia',
  ph: 'asia',
  sa: 'asia',
  ir: 'asia',
  ng: 'africa',
  eg: 'africa',
  gh: 'africa',
  ci: 'africa',
  cm: 'africa',
  sn: 'africa',
  za: 'africa',
  ma: 'africa',
  dz: 'africa',
  ke: 'africa',
}

export function regionForCountry(countryCode?: string): Region {
  if (!countryCode) return 'europe'
  return COUNTRY_REGIONS[countryCode] ?? 'europe'
}

// Weights are parallel to SKIN_TONES above (light -> dark, with one extra light tone at the
// end). Each region has a clear majority complexion but a nonzero chance of every tone - a
// country's population is never actually uniform.
const SKIN_TONE_WEIGHTS: Record<Region, readonly number[]> = {
  europe: [30, 22, 8, 4, 3, 30],
  africa: [3, 4, 10, 25, 40, 3],
  asia: [10, 30, 30, 10, 5, 10],
  americas: [16, 18, 18, 18, 16, 16],
}

// derivePlayerFace is pure: the same seed (a player id works well) always yields the same
// face, so no backend storage is needed and the look survives re-renders and reloads.
// birthCountry (see Player.birthCountry) skews the skin tone toward that region's majority
// complexion - see SKIN_TONE_WEIGHTS.
export function derivePlayerFace(seed: number | string, birthCountry?: string): FaceTraits {
  const rng = createRng(seed)
  const region = regionForCountry(birthCountry)

  let facialHair: FacialHair = 'none'
  if (chance(rng, 0.38)) {
    facialHair = pick(rng, ['mustache', 'goatee', 'stubble'] as const)
  }

  return {
    skin: pickWeighted(rng, SKIN_TONES, SKIN_TONE_WEIGHTS[region]),
    hairStyle: pick(rng, HAIR_STYLES),
    hairColor: pick(rng, HAIR_COLORS),
    browAngle: pick(rng, BROW_ANGLES),
    mouthStyle: pick(rng, MOUTH_STYLES),
    facialHair,
  }
}
