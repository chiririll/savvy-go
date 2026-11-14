interface UserLike {
    email: string
    name?: string
}

const GRAVATAR_SIZE = 160

export async function getUserAvatarUrl(user: UserLike): Promise<string> {
    const data = new TextEncoder().encode(user.email.trim().toLowerCase())
    const digest = await crypto.subtle.digest('SHA-256', data)
    const hash = Array.from(new Uint8Array(digest), (b) => b.toString(16).padStart(2, '0')).join('')
    return `https://www.gravatar.com/avatar/${hash}?s=${GRAVATAR_SIZE}&d=identicon`
}

export function getUserInitials(user: UserLike): string {
    return user.name?.charAt(0).toUpperCase() ?? user.email.charAt(0).toUpperCase()
}
