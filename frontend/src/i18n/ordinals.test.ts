import { describe, expect, it } from 'vitest'
import i18n from '.'

// League positions are ordinals: English needs 1st/2nd/3rd/4th…11th/12th/13th…21st, Spanish
// "2º" for every number (i18next ordinal plurals, Intl.PluralRules({ type: 'ordinal' })).
describe('position ordinals', () => {
  const en = i18n.getFixedT('en')
  const es = i18n.getFixedT('es')

  it.each([
    [1, '1st'],
    [2, '2nd'],
    [3, '3rd'],
    [4, '4th'],
    [11, '11th'],
    [12, '12th'],
    [13, '13th'],
    [21, '21st'],
    [22, '22nd'],
    [23, '23rd'],
    [101, '101st'],
  ])('reads %i as %s in English', (count, text) => {
    expect(en('dashboard.positionValue', { count, ordinal: true })).toBe(text)
    expect(en('subheader.position', { count, ordinal: true })).toBe(text)
    expect(en('preview.position', { count, ordinal: true })).toBe(`Currently ${text} in the league`)
  })

  it('uses the masculine ordinal sign in Spanish', () => {
    expect(es('dashboard.positionValue', { count: 2, ordinal: true })).toBe('2º')
    expect(es('subheader.position', { count: 21, ordinal: true })).toBe('21º')
    expect(es('preview.position', { count: 3, ordinal: true })).toBe('Ahora 3º en la liga')
  })
})
