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

// The continent-sized grouping the international competitions use (see regionForCountry). It no
// longer drives faces - APPEARANCE_PROFILES below is per country group (issue #71 started here).
export type Region = 'europe' | 'africa' | 'asia' | 'americas'

// Mirrors backend common.KnownCountries - every country code the game can currently generate a
// world for. An unrecognized/missing code (legacy players created before birthCountry existed)
// falls back to 'europe'.
export const COUNTRY_REGIONS: Record<string, Region> = {
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
  be: 'europe',
  tr: 'europe',
  at: 'europe',
  sco: 'europe',
  wal: 'europe',
  ie: 'europe',
  ru: 'europe',
  rs: 'europe',
  hr: 'europe',
  ba: 'europe',
  si: 'europe',
  mk: 'europe',
  me: 'europe',
  xk: 'europe',
  ve: 'americas',
  py: 'americas',
  bo: 'americas',
  qa: 'asia',
  ae: 'asia',
  uz: 'asia',
  tn: 'africa',
  tz: 'africa',
  ml: 'africa',
  al: 'europe',
  ad: 'europe',
  am: 'europe',
  az: 'europe',
  by: 'europe',
  bg: 'europe',
  cy: 'europe',
  cz: 'europe',
  dk: 'europe',
  ee: 'europe',
  fo: 'europe',
  fi: 'europe',
  ge: 'europe',
  gi: 'europe',
  hu: 'europe',
  is: 'europe',
  il: 'europe',
  kz: 'europe',
  lv: 'europe',
  li: 'europe',
  lt: 'europe',
  lu: 'europe',
  mt: 'europe',
  md: 'europe',
  nir: 'europe',
  no: 'europe',
  ro: 'europe',
  sm: 'europe',
  sk: 'europe',
  ch: 'europe',
  ua: 'europe',
  ai: 'americas',
  ag: 'americas',
  aw: 'americas',
  bs: 'americas',
  bb: 'americas',
  bz: 'americas',
  bm: 'americas',
  vg: 'americas',
  ky: 'americas',
  cr: 'americas',
  cu: 'americas',
  cw: 'americas',
  dm: 'americas',
  do: 'americas',
  sv: 'americas',
  gd: 'americas',
  gt: 'americas',
  gy: 'americas',
  ht: 'americas',
  hn: 'americas',
  jm: 'americas',
  ms: 'americas',
  ni: 'americas',
  pa: 'americas',
  pr: 'americas',
  kn: 'americas',
  lc: 'americas',
  vc: 'americas',
  sr: 'americas',
  tt: 'americas',
  tc: 'americas',
  vi: 'americas',
  af: 'asia',
  au: 'asia',
  bh: 'asia',
  bd: 'asia',
  bt: 'asia',
  bn: 'asia',
  kh: 'asia',
  tw: 'asia',
  gu: 'asia',
  hk: 'asia',
  iq: 'asia',
  jo: 'asia',
  kw: 'asia',
  kg: 'asia',
  la: 'asia',
  lb: 'asia',
  mo: 'asia',
  my: 'asia',
  mv: 'asia',
  mn: 'asia',
  mm: 'asia',
  np: 'asia',
  kp: 'asia',
  om: 'asia',
  pk: 'asia',
  ps: 'asia',
  sg: 'asia',
  lk: 'asia',
  sy: 'asia',
  tj: 'asia',
  tl: 'asia',
  tm: 'asia',
  ye: 'asia',
  ao: 'africa',
  bj: 'africa',
  bw: 'africa',
  bf: 'africa',
  bi: 'africa',
  cv: 'africa',
  cf: 'africa',
  td: 'africa',
  km: 'africa',
  cg: 'africa',
  cd: 'africa',
  dj: 'africa',
  gq: 'africa',
  er: 'africa',
  sz: 'africa',
  et: 'africa',
  ga: 'africa',
  gm: 'africa',
  gn: 'africa',
  gw: 'africa',
  ls: 'africa',
  lr: 'africa',
  ly: 'africa',
  mg: 'africa',
  mw: 'africa',
  mr: 'africa',
  mu: 'africa',
  mz: 'africa',
  na: 'africa',
  ne: 'africa',
  rw: 'africa',
  st: 'africa',
  sc: 'africa',
  sl: 'africa',
  so: 'africa',
  ss: 'africa',
  sd: 'africa',
  tg: 'africa',
  ug: 'africa',
  zm: 'africa',
  zw: 'africa',
  as: 'asia',
  ck: 'asia',
  fj: 'asia',
  nc: 'asia',
  nz: 'asia',
  pg: 'asia',
  ws: 'asia',
  sb: 'asia',
  pf: 'asia',
  to: 'asia',
  vu: 'asia',
}

export function regionForCountry(countryCode?: string): Region {
  if (!countryCode) return 'europe'
  return COUNTRY_REGIONS[countryCode] ?? 'europe'
}

// A player's look follows their birth country, not just its continent: every country belongs to an
// appearance group (about two dozen, finer than Region - the Gulf, South Asia, the Caribbean or
// Polynesia look nothing like the rest of their continent) with its own odds for skin tone, hair
// colour, hair style and facial hair. Every option stays reachable in every group: a country's
// population is never uniform.
//
// Weights are parallel to SKIN_TONES / HAIR_COLORS / HAIR_STYLES above.
// SKIN_TONES: 0 light, 1 light-medium, 2 medium, 3 brown, 4 dark, 5 very light.
// HAIR_COLORS: 0 black, 1 brown, 2 light brown/blond, 3 red, 4 grey, 5 dark brown.
// HAIR_STYLES: bald, short, side, curly, mohawk.
interface AppearanceProfile {
  skin: readonly number[]
  hairColor: readonly number[]
  hairStyle: readonly number[]
  facialHairChance: number
}

const DARK_HAIR = [60, 10, 1, 1, 3, 25] as const
const STRAIGHT_STYLES = [7, 38, 36, 9, 10] as const
const WAVY_STYLES = [8, 34, 30, 20, 8] as const
const TEXTURED_STYLES = [12, 30, 14, 40, 4] as const

export const APPEARANCE_PROFILES = {
  northEurope: { skin: [30, 12, 4, 2, 2, 50], hairColor: [3, 22, 38, 12, 5, 20], hairStyle: WAVY_STYLES, facialHairChance: 0.4 },
  centralEurope: { skin: [30, 22, 8, 4, 3, 30], hairColor: [8, 28, 24, 6, 5, 29], hairStyle: WAVY_STYLES, facialHairChance: 0.38 },
  southEurope: { skin: [22, 34, 18, 5, 3, 18], hairColor: [26, 30, 10, 2, 4, 28], hairStyle: WAVY_STYLES, facialHairChance: 0.5 },
  eastEurope: { skin: [30, 22, 8, 3, 2, 35], hairColor: [10, 28, 28, 5, 5, 24], hairStyle: WAVY_STYLES, facialHairChance: 0.38 },
  levantTurkic: { skin: [12, 32, 34, 13, 4, 5], hairColor: [40, 18, 3, 1, 4, 34], hairStyle: WAVY_STYLES, facialHairChance: 0.55 },
  arabianMaghreb: { skin: [4, 16, 38, 30, 9, 3], hairColor: DARK_HAIR, hairStyle: WAVY_STYLES, facialHairChance: 0.6 },
  hornOfAfrica: { skin: [2, 5, 15, 38, 38, 2], hairColor: DARK_HAIR, hairStyle: [10, 30, 16, 36, 8], facialHairChance: 0.4 },
  southAsia: { skin: [3, 12, 38, 34, 10, 3], hairColor: DARK_HAIR, hairStyle: WAVY_STYLES, facialHairChance: 0.5 },
  centralAsia: { skin: [14, 36, 30, 10, 3, 7], hairColor: DARK_HAIR, hairStyle: STRAIGHT_STYLES, facialHairChance: 0.35 },
  eastAsia: { skin: [18, 40, 24, 6, 2, 10], hairColor: DARK_HAIR, hairStyle: STRAIGHT_STYLES, facialHairChance: 0.15 },
  southeastAsia: { skin: [6, 24, 44, 20, 4, 2], hairColor: DARK_HAIR, hairStyle: STRAIGHT_STYLES, facialHairChance: 0.2 },
  westAfrica: { skin: [1, 2, 7, 25, 62, 3], hairColor: DARK_HAIR, hairStyle: TEXTURED_STYLES, facialHairChance: 0.34 },
  eastSouthAfrica: { skin: [3, 4, 12, 32, 46, 3], hairColor: DARK_HAIR, hairStyle: TEXTURED_STYLES, facialHairChance: 0.34 },
  southAfrica: { skin: [14, 10, 16, 24, 32, 4], hairColor: [30, 18, 12, 3, 4, 33], hairStyle: [10, 32, 20, 30, 8], facialHairChance: 0.38 },
  latinAndean: { skin: [8, 26, 42, 20, 2, 2], hairColor: DARK_HAIR, hairStyle: STRAIGHT_STYLES, facialHairChance: 0.4 },
  latinMixed: { skin: [14, 20, 26, 22, 14, 4], hairColor: [40, 20, 4, 2, 3, 31], hairStyle: WAVY_STYLES, facialHairChance: 0.45 },
  southernCone: { skin: [24, 32, 24, 7, 3, 10], hairColor: [28, 28, 10, 3, 4, 27], hairStyle: WAVY_STYLES, facialHairChance: 0.42 },
  caribbean: { skin: [2, 4, 12, 28, 50, 4], hairColor: DARK_HAIR, hairStyle: TEXTURED_STYLES, facialHairChance: 0.36 },
  indoCaribbean: { skin: [5, 10, 26, 28, 28, 3], hairColor: DARK_HAIR, hairStyle: WAVY_STYLES, facialHairChance: 0.42 },
  northAmerica: { skin: [16, 18, 18, 18, 16, 16], hairColor: [24, 24, 14, 5, 5, 28], hairStyle: WAVY_STYLES, facialHairChance: 0.38 },
  australasia: { skin: [26, 22, 18, 10, 4, 20], hairColor: [10, 26, 28, 8, 5, 23], hairStyle: WAVY_STYLES, facialHairChance: 0.38 },
  polynesian: { skin: [4, 14, 44, 28, 6, 4], hairColor: DARK_HAIR, hairStyle: [8, 32, 26, 26, 8], facialHairChance: 0.36 },
  melanesian: { skin: [1, 3, 10, 32, 52, 2], hairColor: DARK_HAIR, hairStyle: TEXTURED_STYLES, facialHairChance: 0.34 },
} as const satisfies Record<string, AppearanceProfile>

export type AppearanceGroup = keyof typeof APPEARANCE_PROFILES

const APPEARANCE_GROUP_COUNTRIES: Record<AppearanceGroup, string> = {
  northEurope: 'en sco wal nir ie nl se dk no is fo fi ee lv lt',
  centralEurope: 'de fr at be ch li lu cz sk pl hu',
  southEurope: 'es it pt gr mt cy ad sm al xk gi',
  eastEurope: 'ru ua by md ro bg rs hr ba mk me si',
  levantTurkic: 'tr az ge am il lb sy jo ps ir',
  arabianMaghreb: 'sa qa ae kw bh om ye iq eg ly dz ma tn mr',
  hornOfAfrica: 'sd so dj er et',
  southAsia: 'in pk bd lk np bt mv',
  centralAsia: 'uz kz kg tj tm af',
  eastAsia: 'cn jp kr kp tw hk mo mn',
  southeastAsia: 'th vn id ph my sg bn kh la mm tl',
  westAfrica: 'ng gh ci cm sn ml bf bj tg ne gm gn gw lr sl cv st gq ga cg cd cf td',
  eastSouthAfrica: 'ke tz ug rw bi ss zm mw zw mz ao na bw ls sz mg mu sc km',
  southAfrica: 'za',
  latinAndean: 'mx gt hn sv ni pe bo ec py',
  latinMixed: 'br co ve cu do pa cr pr',
  southernCone: 'ar cl uy',
  caribbean: 'jm ht bb bs gd lc vc ag dm kn ai ms vg ky tc vi bm aw cw bz',
  indoCaribbean: 'gy sr tt',
  northAmerica: 'us ca',
  australasia: 'au nz',
  polynesian: 'ws to ck as pf gu',
  melanesian: 'pg sb vu nc fj',
}

const APPEARANCE_BY_COUNTRY: Record<string, AppearanceGroup> = Object.fromEntries(
  Object.entries(APPEARANCE_GROUP_COUNTRIES).flatMap(([group, codes]) =>
    codes.split(' ').map((code) => [code, group as AppearanceGroup]),
  ),
)

/** The appearance group of a birth country; a missing or unrecognized code (a legacy player created
 * before birthCountry existed) gets central Europe's, the former default. */
export function appearanceForCountry(countryCode?: string): AppearanceGroup {
  return (countryCode && APPEARANCE_BY_COUNTRY[countryCode]) || 'centralEurope'
}

// derivePlayerFace is pure: the same seed (a player id works well) always yields the same
// face, so no backend storage is needed and the look survives re-renders and reloads.
// birthCountry (see Player.birthCountry) skews skin tone, hair colour, hair style and facial hair
// toward what is typical of that country - see APPEARANCE_PROFILES.
export function derivePlayerFace(seed: number | string, birthCountry?: string): FaceTraits {
  const rng = createRng(seed)
  const profile: AppearanceProfile = APPEARANCE_PROFILES[appearanceForCountry(birthCountry)]

  let facialHair: FacialHair = 'none'
  if (chance(rng, profile.facialHairChance)) {
    facialHair = pick(rng, ['mustache', 'goatee', 'stubble'] as const)
  }

  return {
    skin: pickWeighted(rng, SKIN_TONES, profile.skin),
    hairStyle: pickWeighted(rng, HAIR_STYLES, profile.hairStyle),
    hairColor: pickWeighted(rng, HAIR_COLORS, profile.hairColor),
    browAngle: pick(rng, BROW_ANGLES),
    mouthStyle: pick(rng, MOUTH_STYLES),
    facialHair,
  }
}
