import { describe, expect, it } from 'vitest'

// ui-theme "Icon style": the app uses outlined glyphs (filled only for a selected state), never
// MUI's two-tone set. Reads the sources (raw, through Vite) rather than rendered output, so an
// icon in a rarely rendered branch is caught too.
const sources = import.meta.glob<string>(['./**/*.{ts,tsx}', '!./iconStyle.test.ts'], { query: '?raw', import: 'default', eager: true })
const TWO_TONE_IMPORT = /@mui\/icons-material\/[A-Za-z]+TwoTone['"]/

describe('icon style', () => {
  it('reads the app sources', () => {
    expect(Object.keys(sources).length).toBeGreaterThan(100)
  })

  it('imports no two-tone icon (No two-tone icon left)', () => {
    const offenders = Object.entries(sources).flatMap(([file, text]) =>
      text.split('\n').flatMap((line, index) => (TWO_TONE_IMPORT.test(line) ? [`${file}:${index + 1}`] : [])),
    )
    expect(offenders).toEqual([])
  })
})
