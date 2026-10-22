import { useEffect, useRef } from 'react'

interface InfiniteScrollSentinelProps {
    enabled: boolean
    onVisible: () => void
    rootMargin?: string
}

function findScrollParent(element: HTMLElement | null): Element | null {
    let node = element?.parentElement ?? null

    while (node) {
        const overflowY = getComputedStyle(node).overflowY
        if (overflowY === 'auto' || overflowY === 'scroll') {
            return node
        }
        node = node.parentElement
    }

    return element?.closest('main') ?? null
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

        const root = findScrollParent(element)
        const observer = new IntersectionObserver(
            (entries) => {
                if (!entries[0]?.isIntersecting || lockedRef.current) {
                    return
                }

                lockedRef.current = true
                onVisibleRef.current()
            },
            { root, rootMargin, threshold: 0 },
        )

        observer.observe(element)
        return () => observer.disconnect()
    }, [enabled, rootMargin])

    return <div ref={ref} className="h-px w-full" aria-hidden="true" />
}
