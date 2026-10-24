import { ColorIcon } from '@/components/shared/ColorIcon'

interface Props {
    space: { name: string; emoji: string | null; color: string | null }
    className?: string
}

/** The emoji of a space on a tint of its color; the first letter of the name when it has no emoji. */
export function SpaceBadge({ space, className }: Props) {
    const fallback = space.name.trim().charAt(0).toUpperCase() || '?'
    return <ColorIcon emoji={space.emoji} color={space.color} fallback={fallback} className={className} />
}
