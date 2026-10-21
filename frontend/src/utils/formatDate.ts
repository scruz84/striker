import { intlLocale } from './locale'

export function formatDateTime(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) {
    return iso
  }
  const locale = intlLocale()
  return new Intl.DateTimeFormat(locale, {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(date)
}

// Date only, no time - for compact contexts (e.g. HeadToHeadCard's past-meetings list) where the
// full formatDateTime's extra ", HH:mm" just adds width without adding anything a viewer skimming
// a list of results needs.
export function formatDate(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) {
    return iso
  }
  const locale = intlLocale()
  return new Intl.DateTimeFormat(locale, { day: '2-digit', month: '2-digit', year: '2-digit' }).format(date)
}

// Day/month plus kickoff time, no year - for a list scoped to the current season (e.g.
// TeamMatchesSection's row) where the year is implied and formatDateTime's full "DD/MM/YYYY,
// HH:mm" is wider than a narrow row has room for.
export function formatDayTime(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) {
    return iso
  }
  const locale = intlLocale()
  return new Intl.DateTimeFormat(locale, { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false }).format(date)
}

// Day/month with no time or year - the shortest unambiguous form, for table cells and two-line
// list rows on a phone ("20/10").
export function formatShortDate(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) {
    return iso
  }
  return new Intl.DateTimeFormat(intlLocale(), { day: '2-digit', month: '2-digit' }).format(date)
}
