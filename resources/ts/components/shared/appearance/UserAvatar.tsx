import { useEffect, useState } from 'react'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { getUserAvatarUrl, getUserInitials } from '@/lib/avatar'

interface UserAvatarProps {
    user: { email: string; name?: string }
    className?: string
}

/** The avatar as drawn: the picture when there is one, otherwise the initial. */
export function UserAvatarView({ user, src, className }: UserAvatarProps & { src?: string }) {
    return (
        <Avatar className={className}>
            {src && <AvatarImage src={src} alt={user.name} />}
            <AvatarFallback>{getUserInitials(user)}</AvatarFallback>
        </Avatar>
    )
}

export function UserAvatar({ user, className }: UserAvatarProps) {
    const [src, setSrc] = useState<string>()

    useEffect(() => {
        let cancelled = false
        getUserAvatarUrl(user)
            .then((url) => {
                if (!cancelled) setSrc(url)
            })
            .catch(() => {})
        return () => {
            cancelled = true
        }
    }, [user.email])

    return <UserAvatarView user={user} src={src} className={className} />
}
