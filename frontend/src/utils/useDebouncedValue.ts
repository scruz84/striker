import { useEffect, useState } from 'react'

// Returns `value`, but only after it has stopped changing for `delayMs` - so a text field can drive
// a server search without firing one request per keystroke.
export function useDebouncedValue<T>(value: T, delayMs: number): T {
  const [debounced, setDebounced] = useState(value)

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delayMs)
    return () => clearTimeout(timer)
  }, [value, delayMs])

  return debounced
}
