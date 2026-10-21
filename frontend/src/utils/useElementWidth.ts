import { useEffect, useRef, useState } from 'react'

/** Tracks an element's rendered width so an SVG can be drawn in real pixels instead of a fixed
 * viewBox that the browser shrinks - which is what made chart labels come out at 7px on a phone. */
export function useElementWidth<T extends HTMLElement>(fallback = 320) {
  const ref = useRef<T>(null)
  const [width, setWidth] = useState(fallback)

  useEffect(() => {
    const element = ref.current
    if (!element) {
      return
    }
    setWidth(element.clientWidth || fallback)
    if (typeof ResizeObserver === 'undefined') {
      return
    }
    const observer = new ResizeObserver(([entry]) => setWidth(Math.round(entry.contentRect.width) || fallback))
    observer.observe(element)
    return () => observer.disconnect()
  }, [fallback])

  return { ref, width }
}
