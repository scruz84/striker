import type { TFunction } from 'i18next'

interface LedgerPattern {
  pattern: RegExp
  key: string
  /** Maps the regex groups onto the translation's interpolation params. */
  params?: (match: RegExpExecArray) => Record<string, string | number>
}

// The backend stores ledger descriptions as English sentences (see engine/wages.go, stadium.go,
// prizemoney.go, common/offers.go…). Until it returns a code + parameters, the known shapes are
// recognised here and rendered in the UI language - including rows written before this existed.
// Anything unrecognised (an admin's free-text adjustment, say) is shown as stored.
const PATTERNS: LedgerPattern[] = [
  { pattern: /^matchday wages$/i, key: 'finances.ledger.matchdayWages' },
  { pattern: /^stadium maintenance$/i, key: 'finances.ledger.stadiumMaintenance' },
  { pattern: /^Club reset after owner left$/i, key: 'finances.ledger.teamReset' },
  { pattern: /^wage contribution for loaned-out players$/i, key: 'finances.ledger.loanWageContribution' },
  { pattern: /^trainer wages: (.+)$/i, key: 'finances.ledger.trainerWages', params: (m) => ({ name: m[1] }) },
  { pattern: /^Matchday ticket revenue \((\d+) attendance\)$/i, key: 'finances.ledger.ticketRevenue', params: (m) => ({ attendance: Number(m[1]) }) },
  {
    pattern: /^Friendly ticket revenue \((\d+) attendance, half share\)$/i,
    key: 'finances.ledger.friendlyTicketRevenue',
    params: (m) => ({ attendance: Number(m[1]) }),
  },
  { pattern: /^Matchday sponsorship$/i, key: 'finances.ledger.sponsorship' },
  { pattern: /^Merchandising revenue$/i, key: 'finances.ledger.merchandising' },
  { pattern: /^Season ticket sales$/i, key: 'finances.ledger.seasonTickets' },
  { pattern: /^(\d+)(?:st|nd|rd|th) place finish$/i, key: 'finances.ledger.placeFinish', params: (m) => ({ position: Number(m[1]) }) },
  { pattern: /^Cup round (\d+) prize money$/i, key: 'finances.ledger.cupRoundPrize', params: (m) => ({ round: Number(m[1]) }) },
  { pattern: /^Cup champion bonus$/i, key: 'finances.ledger.cupChampionBonus' },
  { pattern: /^Stadium expansion to (\d+) seats under construction$/i, key: 'finances.ledger.stadiumExpansion', params: (m) => ({ seats: Number(m[1]) }) },
  { pattern: /^Bought (.+?) from (.+)$/i, key: 'finances.ledger.bought', params: (m) => ({ player: m[1], team: m[2] }) },
  { pattern: /^Sold (.+?) to (.+)$/i, key: 'finances.ledger.sold', params: (m) => ({ player: m[1], team: m[2] }) },
  { pattern: /^Released (.+)$/i, key: 'finances.ledger.released', params: (m) => ({ player: m[1] }) },
  { pattern: /^Paid free agent transfer premium for (.+)$/i, key: 'finances.ledger.premiumPaid', params: (m) => ({ player: m[1] }) },
  { pattern: /^Received free agent transfer premium for (.+)$/i, key: 'finances.ledger.premiumReceived', params: (m) => ({ player: m[1] }) },
  { pattern: /^Loan fee for (.+?) to (.+)$/i, key: 'finances.ledger.loanFeePaid', params: (m) => ({ player: m[1], team: m[2] }) },
  { pattern: /^Loan fee for (.+?) from (.+)$/i, key: 'finances.ledger.loanFeeReceived', params: (m) => ({ player: m[1], team: m[2] }) },
  { pattern: /^Exercised buy option for (.+?) from (.+)$/i, key: 'finances.ledger.buyOptionExercised', params: (m) => ({ player: m[1], team: m[2] }) },
  { pattern: /^Buy option exercised for (.+?) by (.+)$/i, key: 'finances.ledger.buyOptionReceived', params: (m) => ({ player: m[1], team: m[2] }) },
]

export function translateLedgerDescription(t: TFunction, description: string): string {
  for (const { pattern, key, params } of PATTERNS) {
    const match = pattern.exec(description)
    if (match) {
      return t(key, params ? params(match) : {})
    }
  }
  return description
}

/** Cup names arrive from the backend as English ("Cup", "European Cup"…); known ones are translated. */
const CUP_NAME_KEYS: Record<string, string> = {
  Cup: 'cup.names.national',
  'European Cup': 'cup.names.europe',
  'Asian Cup': 'cup.names.asia',
  'African Cup': 'cup.names.africa',
  'Copa America': 'cup.names.americas',
  'International Cup': 'cup.names.international',
}

export function translateCupName(t: TFunction, name: string): string {
  const key = CUP_NAME_KEYS[name]
  return key ? t(key) : name
}
