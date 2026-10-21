import type { TFunction } from 'i18next'
import { parseDivisionName } from '../app/navigation'

/** Readable name for a division: "I.1" -> "1ª División", "III.16" -> "3ª División · Grupo 16".
 * A single-group top division shows no group; any custom name is returned verbatim. */
export function formatDivisionName(t: TFunction, name: string, options: { short?: boolean } = {}): string {
  const parsed = parseDivisionName(name)
  if (!parsed) {
    return name
  }
  if (options.short || parsed.group === undefined) {
    return t(options.short ? 'divisionName.tierShort' : 'divisionName.tier', { tier: parsed.tier })
  }
  return t('divisionName.tierGroup', { tier: parsed.tier, group: parsed.group })
}
