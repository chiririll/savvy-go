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

/** A new invitation. The link is always returned; the email is a convenience that may fail. */
export interface InviteResult {
    token: string
    emailSent: boolean
    /** Why the email was not sent; absent when mail is off or the invitation has no address. */
    emailError?: string
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

export const SYNC_KINDS = ['categories', 'tags'] as const
export type SyncKind = (typeof SYNC_KINDS)[number]

/** The sync group of a space: spaces that keep their categories and/or tags identical. */
export interface SyncGroup {
    id: number
    categories: boolean
    tags: boolean
    /** The group's spaces the caller belongs to. */
    members: (LinkedSpace & { role: SpaceRole })[]
    /** Spaces of the group the caller does not belong to. */
    hidden: number
}

export interface SyncCategoryState {
    spaceId: number
    icon: string | null
    color: string | null
    isDefault: boolean
    transactions: number
}

export interface SyncCategoryConflict {
    type: 'income' | 'expense'
    name: string
    present: SyncCategoryState[]
    missing: number[]
}

export interface SyncTagConflict {
    name: string
    present: { spaceId: number; transactions: number }[]
    missing: number[]
}

export interface SyncDefaultConflict {
    type: 'income' | 'expense'
    defaults: { spaceId: number; name: string }[]
}

/** What makes the spaces of a group differ, for the caller to decide. */
export interface SyncDiff {
    categories: SyncCategoryConflict[]
    tags: SyncTagConflict[]
    defaults: SyncDefaultConflict[]
    spaces: LinkedSpace[]
}

/** A change of a group; `categories` and `tags` are what it keeps identical afterwards. */
export interface SyncChange {
    group_id?: number
    add_space_ids?: number[]
    remove_space_ids?: number[]
    categories: boolean
    tags: boolean
}

export interface SyncDecision {
    kind: SyncKind
    type?: string
    name: string
    keep: boolean
    /** The space whose icon and color win. */
    from_space_id?: number
    /** Name of the category that takes the transactions of a deleted one. */
    successor?: string
}

export interface SyncPlan extends SyncChange {
    decisions: SyncDecision[]
    defaults: { type: string; name: string }[]
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
