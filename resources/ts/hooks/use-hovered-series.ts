import { useCallback, useMemo, useRef } from 'react'

/**
 * Tracks which series line is under the cursor. A ref, not state: the axis tooltip reads
 * it on each refresh, so hovering never rebuilds the chart option.
 */
export function useHoveredSeries() {
    const hovered = useRef<number | null>(null)
    const getHoveredIndex = useCallback(() => hovered.current, [])
    const events = useMemo(
        () => ({
            mouseover: (params: { seriesIndex?: number }) => {
                hovered.current = params.seriesIndex ?? null
            },
            mouseout: () => {
                hovered.current = null
            },
        }),
        [],
    )
    return { getHoveredIndex, events }
}
