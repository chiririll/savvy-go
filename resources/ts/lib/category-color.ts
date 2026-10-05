import type { CSSProperties } from 'react'

/**
 * How a category's colour backs its emoji wherever it is shown: a tint of
 * the colour, strong enough to tell colours apart yet keep the emoji legible.
 */
export function categoryIconStyle(color?: string | null): CSSProperties | undefined {
    if (!color) {
        return undefined
    }
    return { backgroundColor: `${color}33` }
}
