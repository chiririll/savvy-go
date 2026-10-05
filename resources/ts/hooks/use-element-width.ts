import { useLayoutEffect, useState } from 'react'

/** Charts narrower than this (px) switch to compact layouts. */
export const NARROW_CHART_WIDTH = 520

/**
 * Tracks an element's content width. Returns a callback ref (so it also works
 * for elements that mount later, e.g. after data loads) and the current width,
 * which is 0 until the element is measured.
 */
export function useElementWidth<T extends HTMLElement>() {
    const [node, setNode] = useState<T | null>(null)
    const [width, setWidth] = useState(0)

    useLayoutEffect(() => {
        if (!node) return
        setWidth(node.clientWidth)
        const observer = new ResizeObserver(([entry]) => setWidth(Math.round(entry.contentRect.width)))
        observer.observe(node)
        return () => observer.disconnect()
    }, [node])

    return [setNode, width] as const
}

export function isNarrowChart(width: number) {
    return width > 0 && width < NARROW_CHART_WIDTH
}
