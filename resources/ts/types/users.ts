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
    createdAt: string
    token?: string
    expiresAt?: string
}

export interface PasswordTokenPreview {
    name: string
    email: string
    isInactive: boolean
    expiresAt: string
}
