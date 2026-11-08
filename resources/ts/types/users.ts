/** Server role; what a user may do with a space's data is their role in that space. */
export const USER_ROLES = ['admin', 'user', 'guest'] as const
export type UserRole = (typeof USER_ROLES)[number]

export interface User {
    id: number
    name: string
    email: string
    role: UserRole
    isInactive: boolean
    isSsoOnly: boolean
    emailVerified: boolean
    createdAt: string
    /** When a session or API token of the user was last used; null if never. */
    lastActiveAt: string | null
    token?: string
    expiresAt?: string
    /** Set when the reset link was emailed instead of returned as `token`. */
    emailSent?: boolean
    /** Set when a new user was created but the link could not be emailed. */
    emailError?: string
}

export interface PasswordTokenPreview {
    name: string
    email: string
    isInactive: boolean
    expiresAt: string
}
