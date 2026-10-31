import { describe, expect, it } from 'vitest'
import { isDeleteConfirmation } from './deleteConfirmation'

describe('isDeleteConfirmation', () => {
  it('needs the exact club name when the user has a club', () => {
    expect(isDeleteConfirmation('CD Norte', 'CD Norte')).toBe(true)
    expect(isDeleteConfirmation('  CD Norte  ', 'CD Norte')).toBe(true)
    expect(isDeleteConfirmation('cd norte', 'CD Norte')).toBe(false)
    expect(isDeleteConfirmation('DELETE', 'CD Norte')).toBe(false)
    expect(isDeleteConfirmation('', 'CD Norte')).toBe(false)
  })

  it('accepts DELETE or BORRAR in any case when the user has no club', () => {
    for (const word of ['DELETE', 'delete', 'BORRAR', 'Borrar']) {
      expect(isDeleteConfirmation(word, '')).toBe(true)
    }
    expect(isDeleteConfirmation('yes', '')).toBe(false)
    expect(isDeleteConfirmation('', '')).toBe(false)
  })
})
