import { vi } from 'vitest'

/** Stubs window.matchMedia so `min-width`/`max-width` queries (as MUI's useMediaQuery and
 * breakpoints emit them) answer for a window `width` px wide. Undo with vi.unstubAllGlobals(). */
export function stubViewportWidth(width: number) {
  const matches = (query: string) => {
    const min = /min-width:\s*([\d.]+)px/.exec(query)
    const max = /max-width:\s*([\d.]+)px/.exec(query)
    return (!min || width >= Number(min[1])) && (!max || width <= Number(max[1])) && (!!min || !!max)
  }
  vi.stubGlobal('matchMedia', (query: string) => ({
    matches: matches(query),
    media: query,
    onchange: null,
    addEventListener: () => {},
    removeEventListener: () => {},
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => false,
  }))
}
