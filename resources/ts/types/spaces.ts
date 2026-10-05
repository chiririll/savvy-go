export const SPACE_ROLES = ['admin', 'editor', 'viewer'] as const
export type SpaceRole = (typeof SPACE_ROLES)[number]

export interface Space {
    id: number
    uuid: string
    name: string
    role: SpaceRole
}

export interface SpaceDetails extends Space {
    /** Bytes; 0 means unlimited. */
    quota: number
    size: number
}

export interface SpaceMember {
    userId: number
    name: string
    email: string
    role: SpaceRole
    createdAt: string | null
}

export interface SpaceInvitation {
    id: number
    email: string | null
    role: SpaceRole
    inviter: string
    expiresAt: string | null
    acceptedAt: string | null
    createdAt: string | null
}

export interface InvitationPreview {
    spaceName: string
    role: SpaceRole
    inviter: string
    email: string | null
    expiresAt: string
    canRegister: boolean
}

export interface LinkedSpace {
    id: number
    name: string
}

export interface TransferSide {
    amount: string
    currency: string
    /** Only for members of the space this side belongs to. */
    accountId?: number
}

export const TRANSFER_REVIEWS = ['created_remote', 'deleted_remote', 'changed_remote', 'missing_remote'] as const
export type TransferReview = (typeof TRANSFER_REVIEWS)[number]

export interface SpaceTransfer {
    uuid: string
    from: TransferSide
    to: TransferSide
    outgoing: boolean
    date: string
    description: string | null
    version: number
    status: 'ok' | 'needs_attention' | 'pending'
    review: TransferReview | null
    /** The other space is no longer linked here. */
    frozen: boolean
    /** Signed by this server or a key it trusts. */
    verified: boolean
    deleted: boolean
    otherSpace?: { id: number; name: string }
    remote?: { from: TransferSide; to: TransferSide; date: string; description: string | null; deleted: boolean }
}

export interface SpaceTransferInput {
    to_space_id: number
    from_account_id: number
    to_account_id: number
    from_amount: string
    to_amount: string
    date: string
    description?: string | null
}

export interface AuditEntry {
    id: number
    actor: string
    action: string
    targetUserId: number | null
    details: string
    createdAt: string | null
}

export interface SpaceOverview {
    id: number
    uuid: string
    name: string
    members: number
    admins: number
    size: number
    quota: number
    unavailable: string
    createdAt: string | null
}

export interface SigningKeys {
    own: { kid: string; publicKey: string }
    trusted: { kid: string; publicKey: string; name: string; createdAt: string | null }[]
}
