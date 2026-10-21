import { intlLocale } from './locale'

// The game has a single currency, whatever the UI language - only the separators and the
// symbol's position follow the active locale (`400.000 €` in Spanish, `€400,000` in English).
const GAME_CURRENCY = 'EUR'

export interface FormatCurrencyOptions {
  /** Abbreviated form for KPIs and narrow rows (`400 k €`, `1,2 M €`). */
  compact?: boolean
  /** Always show the sign of a non-zero amount (`+1.200 €`) - for ledgers and deltas. */
  signed?: boolean
}

export function formatCurrency(amount: number, options: FormatCurrencyOptions = {}): string {
  return new Intl.NumberFormat(intlLocale(), {
    style: 'currency',
    currency: GAME_CURRENCY,
    maximumFractionDigits: options.compact ? 1 : 0,
    notation: options.compact ? 'compact' : 'standard',
    signDisplay: options.signed ? 'exceptZero' : 'auto',
  }).format(amount)
}

export function formatNumber(value: number, options: { compact?: boolean } = {}): string {
  return new Intl.NumberFormat(intlLocale(), {
    maximumFractionDigits: options.compact ? 1 : 0,
    notation: options.compact ? 'compact' : 'standard',
  }).format(value)
}

/** Currency symbol for the game currency in the active locale - used as an input adornment. */
export function currencySymbol(): string {
  const part = new Intl.NumberFormat(intlLocale(), { style: 'currency', currency: GAME_CURRENCY })
    .formatToParts(0)
    .find((p) => p.type === 'currency')
  return part?.value ?? '€'
}
