import { useEffect, useRef, useState } from 'react'

/**
 * Reports whether an element's content overflows its box, and keeps that up to date
 * as the element resizes. Used to decide whether a one-line summary needs a button
 * to reveal the rest, instead of guessing from the text length.
 */
export function useIsTruncated<T extends HTMLElement>() {
  const ref = useRef<T | null>(null)
  const [isTruncated, setIsTruncated] = useState(false)

  useEffect(() => {
    const element = ref.current
    if (!element) return

    const measure = () => {
      // A pixel of slack: sub-pixel layout can report a 0.5px overflow on text
      // that visually fits.
      setIsTruncated(element.scrollWidth > element.clientWidth + 1)
    }

    measure()

    // ResizeObserver watches the box, not the content: a web font applying after
    // mount changes the text width without resizing the element, so re-measure once
    // fonts are settled too.
    let cancelled = false
    document.fonts?.ready.then(() => {
      if (!cancelled) measure()
    })

    const observer = new ResizeObserver(measure)
    observer.observe(element)

    return () => {
      cancelled = true
      observer.disconnect()
    }
  }, [])

  return { ref, isTruncated }
}
