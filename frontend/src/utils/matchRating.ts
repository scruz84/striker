// Most players cluster around the 6.0 base rating (see backend common.ComputeMatchRatings) -
// only a genuinely strong or poor showing should read as green/red, so "default" (a neutral
// gray) covers that whole unremarkable middle band rather than warning-orange swallowing it.
export function matchRatingColor(rating: number): 'success' | 'error' | 'default' {
  if (rating >= 7.5) return 'success'
  if (rating < 4) return 'error'
  return 'default'
}
