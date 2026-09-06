// Tiny deterministic PRNG so a player's face is stable across renders/sessions without the
// backend needing to store anything - the same seed always produces the same face.

// xmur3 string hash, used to turn a seed of any shape into a well-mixed 32-bit int.
function hashToSeed(input: string): number {
  let h = 1779033703 ^ input.length
  for (let i = 0; i < input.length; i++) {
    h = Math.imul(h ^ input.charCodeAt(i), 3432918353)
    h = (h << 13) | (h >>> 19)
  }
  return h >>> 0
}

// mulberry32 - small, fast, good enough distribution for picking cosmetic traits.
function mulberry32(seed: number) {
  let a = seed
  return () => {
    a |= 0
    a = (a + 0x6d2b79f5) | 0
    let t = Math.imul(a ^ (a >>> 15), 1 | a)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

export type Rng = () => number

export function createRng(seed: number | string): Rng {
  return mulberry32(typeof seed === 'number' ? seed : hashToSeed(seed))
}

export function pick<T>(rng: Rng, options: readonly T[]): T {
  return options[Math.floor(rng() * options.length) % options.length]
}

export function chance(rng: Rng, probability: number): boolean {
  return rng() < probability
}
