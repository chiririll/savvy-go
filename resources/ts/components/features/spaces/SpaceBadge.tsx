import { cn } from '@/lib/utils'

interface Props {
    space: { name: string; emoji: string | null; color: string | null }
    className?: string
}

/** The emoji of a space on its color; the first letter of the name when it has no emoji. */
export function SpaceBadge({ space, className }: Props) {
    const label = space.emoji || space.name.trim().charAt(0).toUpperCase() || '?'
    return (
        <div
            aria-hidden
            className={cn(
                'flex aspect-square size-8 shrink-0 items-center justify-center rounded-lg text-sm font-semibold',
                space.color ? 'text-white' : 'bg-primary text-primary-foreground',
                className,
            )}
            style={space.color ? { backgroundColor: space.color } : undefined}
        >
            {label}
        </div>
    )
}
