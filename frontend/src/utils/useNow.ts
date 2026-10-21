import { useEffect, useState } from 'react'

/** Current time, refreshed every `intervalMs` - for countdowns that should tick without a reload. */
export function useNow(intervalMs = 30000): number {
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    const handle = window.setInterval(() => setNow(Date.now()), intervalMs)
    return () => window.clearInterval(handle)
  }, [intervalMs])
  return now
}
