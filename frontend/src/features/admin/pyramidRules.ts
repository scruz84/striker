import type { TierConfig } from '../../api/types'

// The league pyramid rule the backend enforces on a saved world structure (backend
// common.ValidateWorldConfig, openspec/specs/league-pyramid "Pyramid structure rule"), mirrored
// here so the editor can explain what is wrong before saving. The backend stays the authority.
export const MAX_WORLD_TIERS = 6
export const MIN_TEAMS_PER_LEAGUE = 8
export const MAX_TEAMS_PER_LEAGUE = 24
export const MAX_TEAMS_PER_COUNTRY = 1500

export interface PyramidError {
  key: string
  params?: Record<string, number | string>
}

// pyramidError returns the first rule tiers break, or null for a valid pyramid.
export function pyramidError(tiers: TierConfig[]): PyramidError | null {
  if (tiers.length < 1 || tiers.length > MAX_WORLD_TIERS) {
    return { key: 'admin.worldStructure.errors.tierCount', params: { max: MAX_WORLD_TIERS } }
  }
  const byTier = new Map(tiers.map((tier) => [tier.tier, tier]))
  for (let tier = 1; tier <= tiers.length; tier++) {
    if (!byTier.has(tier)) return { key: 'admin.worldStructure.errors.gaps', params: { count: tiers.length } }
  }
  for (const tier of tiers) {
    if (!tier.name.trim()) return { key: 'admin.worldStructure.errors.name', params: { tier: tier.tier } }
    if (tier.teamsPerLeague < MIN_TEAMS_PER_LEAGUE || tier.teamsPerLeague > MAX_TEAMS_PER_LEAGUE || tier.teamsPerLeague % 2 !== 0) {
      return { key: 'admin.worldStructure.errors.teams', params: { tier: tier.tier, min: MIN_TEAMS_PER_LEAGUE, max: MAX_TEAMS_PER_LEAGUE } }
    }
  }
  if (byTier.get(1)?.groups !== 1) return { key: 'admin.worldStructure.errors.topTier' }
  for (let tier = 2; tier <= tiers.length; tier++) {
    const want = (byTier.get(tier - 1)?.groups ?? 0) * 2
    if (byTier.get(tier)?.groups !== want) {
      return { key: 'admin.worldStructure.errors.doubling', params: { tier, groups: want, upper: tier - 1 } }
    }
    if (byTier.get(tier)?.teamsPerLeague !== byTier.get(1)?.teamsPerLeague) {
      return { key: 'admin.worldStructure.errors.sameTeams' }
    }
  }
  const places = worldPlaces(tiers)
  if (places > MAX_TEAMS_PER_COUNTRY) {
    return { key: 'admin.worldStructure.errors.tooManyTeams', params: { count: places, max: MAX_TEAMS_PER_COUNTRY } }
  }
  return null
}

// worldPlaces is how many teams one country holds under tiers.
export function worldPlaces(tiers: TierConfig[]): number {
  return tiers.reduce((sum, tier) => sum + tier.groups * tier.teamsPerLeague, 0)
}
