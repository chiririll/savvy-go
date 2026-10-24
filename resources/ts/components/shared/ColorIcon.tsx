import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface Props {
    emoji?: string | null
    /** Hex colour. The icon is drawn on a tint of it. */
    color?: string | null
    /** Shown when there is no emoji and no icon, for example the first letter of a name. */
    fallback?: string
    /** Shown when there is no emoji: a glyph or any node, for example the icon of a transaction type. */
    icon?: ReactNode
    /** Classes for the box when there is no colour: the background and text colour of a neutral tone. */
    tone?: string
    /** `md` is 32px, `lg` is 40px. */
    size?: 'md' | 'lg'
    shape?: 'square' | 'circle'
    /** Hides the box from assistive technology: set it when the text beside it says the same. */
    decorative?: boolean
    className?: string
}

const SIZES = {
    md: 'size-8 text-sm font-semibold',
    lg: 'size-10 text-base',
} as const

/**
 * The one place that decides how an emoji on a colour looks: the icon of a
 * space, category, budget or transaction. The colour is a tint, so the text
 * colour follows the theme and stays legible in light and dark.
 */
export function ColorIcon({
    emoji,
    color,
    fallback,
    icon,
    tone,
    size = 'md',
    shape = 'square',
    decorative,
    className,
}: Props) {
    return (
        <div
            aria-hidden={decorative || undefined}
            className={cn(
                'flex aspect-square shrink-0 items-center justify-center',
                SIZES[size],
                shape === 'circle' ? 'rounded-full' : 'rounded-lg',
                color ? 'text-foreground' : (tone ?? 'bg-muted text-foreground'),
                className,
            )}
            style={color ? { backgroundColor: `${color}33` } : undefined}
        >
            {emoji || icon || fallback}
        </div>
    )
}
