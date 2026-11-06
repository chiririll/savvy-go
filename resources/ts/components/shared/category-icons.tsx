import type { ReactNode } from 'react'
import type { Category } from '@/types'

/**
 * The emoji of one category, or two of them stacked corner to corner for several. Null when none has
 * an emoji, so the caller draws its own icon.
 */
export function categoryIcons(categories: Array<Pick<Category, 'icon'>>): ReactNode {
    const icons = categories
        .map((category) => category.icon)
        .filter(Boolean)
        .slice(0, 2)
    if (icons.length === 0) {
        return null
    }
    if (icons.length === 1) {
        return <>{icons[0]}</>
    }
    return (
        <span className="relative block size-full text-sm">
            <span className="absolute left-1 top-1">{icons[0]}</span>
            <span className="absolute bottom-1 right-1">{icons[1]}</span>
        </span>
    )
}
