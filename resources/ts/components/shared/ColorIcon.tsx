import { categoryIconStyle } from '@/lib/category-color'
import { cn } from '@/lib/utils'

interface Props {
    emoji?: string | null
    color?: string | null
    /** Shown when there is no emoji, for example the first letter of a name. */
    fallback?: string
    className?: string
}

/**
 * The icon of a space or category: the emoji on a tint of its colour. The text
 * colour follows the theme, so it stays legible in light and dark.
 */
export function ColorIcon({ emoji, color, fallback, className }: Props) {
    const style = categoryIconStyle(color)
    return (
        <div
            aria-hidden
            className={cn(
                'flex aspect-square size-8 shrink-0 items-center justify-center rounded-lg text-sm font-semibold text-foreground',
                !style && 'bg-muted',
                className,
            )}
            style={style}
        >
            {emoji || fallback}
        </div>
    )
}
