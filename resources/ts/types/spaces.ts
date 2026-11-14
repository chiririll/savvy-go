export const SPACE_ROLES = ['admin', 'editor', 'viewer'] as const
export type SpaceRole = (typeof SPACE_ROLES)[number]

export interface Space {
    id: number
    uuid: string
    name: string
    /** One emoji, or null. */
    emoji: string | null
    /** #rrggbb, or null. */
    color: string | null
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
    /** When the member last used the app or an API token; null if never. */
    lastActiveAt: string | null
}

export interface SpaceInvitation {
    id: number
    email: string | null
    /** Who a link invitation is for, as its inviter wrote it. */
    note: string | null
    role: SpaceRole
    inviter: string
    expiresAt: string | null
    acceptedAt: string | null
    createdAt: string | null
}

/**
 * A new invitation. One made with an address is emailed and has no link here:
 * only the mailbox gets it. One made without an address returns the link to copy.
 */
export interface InviteResult {
    id: number
    token?: string
    emailSent: boolean
}

/** How the server lets invitations be made right now. */
export interface InviteMethods {
    /** By email: needs working mail. */
    email: boolean
    /** By a link to copy: not allowed while verified addresses are required. */
    link: boolean
    /** Why email is off, when it is. */
    emailReason?: string
}

export interface InvitationPreview {
    /** The signed-in viewer is in the space already, so the link is of no use to them. */
    alreadyMember: boolean
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

/** `space_transfers.status`: a transfer in review waits for the owner to pick a side. */
export const TRANSFER_STATUSES = ['confirmed', 'in_review'] as const
export type TransferStatus = (typeof TRANSFER_STATUSES)[number]

export interface SpaceTransfer {
    uuid: string
    from: TransferSide
    to: TransferSide
    outgoing: boolean
    date: string
    description: string | null
    version: number
    status: TransferStatus
    review: TransferReview | null
    /** Confirmed, but this space's account is gone, so there is no transaction for it. */
    needsAttention: boolean
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

/** A row of the space's change log (`GET /spaces/{id}/audit-log`). Values are open-ended: show unknown ones as they are. */
export interface SpaceAction {
    id: number
    at: string
    /** The door the change came through: ui, api, mcp, or system. */
    source: string
    userId: number | null
    tokenLabel?: string
    entityType: string
    entityId: number | null
    /** create, update, delete, confirm, skip, rollback, cleanup. */
    op: string
    /** Entries of one bulk call (seeding, MCP bulk create) share it. */
    batchId?: string
    details?: string
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
    emoji: string | null
    color: string | null
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

/** The categories and tags a new space can start with (GET /catalog/presets). */
export interface SpacePresets {
    categories: {
        name: string
        type: 'income' | 'expense'
        icon: string
        color: string
        isDefault: boolean
    }[]
    tags: string[]
}

/** The body of PATCH /spaces/{id}; an empty emoji or color clears it. */
export interface SpaceUpdate {
    name: string
    emoji: string
    color: string
}

/** Optional content of POST /spaces beside the name. */
export interface CreateSpaceInput {
    name: string
    emoji?: string
    color?: string
    baseCurrency?: string
    currencies?: string[]
    accounts?: { name: string; type?: string; currency: string; initialBalance?: string }[]
    categories?: SpacePresets['categories']
    tags?: string[]
}
