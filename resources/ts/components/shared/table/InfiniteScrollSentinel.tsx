import { useEffect, useRef } from 'react'

interface InfiniteScrollSentinelProps {
    enabled: boolean
    onVisible: () => void
    rootMargin?: string
}

export function InfiniteScrollSentinel({
    enabled,
    onVisible,
    rootMargin = '240px',
}: InfiniteScrollSentinelProps) {
    const ref = useRef<HTMLDivElement>(null)
    const onVisibleRef = useRef(onVisible)
    const lockedRef = useRef(false)
    onVisibleRef.current = onVisible

    useEffect(() => {
        const element = ref.current
        if (!element || !enabled) {
            lockedRef.current = false
            return
        }

        const observer = new IntersectionObserver(
            (entries) => {
                if (!entries[0]?.isIntersecting || lockedRef.current) {
                    return
                }

                lockedRef.current = true
                onVisibleRef.current()
            },
            // Viewport root: the observer already clips by every scrolling ancestor. A wrapper
            // with overflow-y-auto but no fixed height grows with its content and never scrolls;
            // using it as root made the sentinel always visible and loaded every page at once.
            { root: null, rootMargin, threshold: 0 },
        )

        observer.observe(element)
        return () => observer.disconnect()
    }, [enabled, rootMargin])

    return <div ref={ref} className="h-px w-full" aria-hidden="true" />
}
