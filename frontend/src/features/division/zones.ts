import type { PyramidZones } from '../../api/types'

export type Zone = 'promotion' | 'playoff' | 'relegation' | undefined

// Which zones a table's legend lists: only the ones the league has. `candidate` relabels the
// playoff colour "Promotion candidate" for a legacy pyramid, where a group winner only goes up if
// it is among the best winners of its tier.
export interface ZoneLegendFlags {
  promotion?: boolean
  playoff?: boolean
  relegation?: boolean
  candidate?: boolean
}

// zoneFor maps a current-season position onto the league pyramid's zones (backend common.ZoneOf):
// direct promotion, the promotion/relegation playoff (or a legacy promotion candidate), direct
// relegation. The table never marks more teams as promoted than actually go up.
export function zoneFor(position: number, total: number, zones: PyramidZones | undefined): Zone {
  if (!zones || position < 1 || position > total) return undefined
  if (position <= zones.directUp) return 'promotion'
  if (position <= zones.directUp + zones.playoffUp + zones.candidate) return 'playoff'
  if (position > total - zones.directDown) return 'relegation'
  if (position > total - zones.directDown - zones.playoffDown) return 'playoff'
  return undefined
}

export function zonesLegend(zones: PyramidZones | undefined): ZoneLegendFlags {
  if (!zones) return {}
  return {
    promotion: zones.directUp > 0,
    playoff: zones.playoffUp + zones.playoffDown + zones.candidate > 0,
    relegation: zones.directDown > 0,
    candidate: zones.legacy && zones.candidate > 0,
  }
}
