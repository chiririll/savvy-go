import type { InviteMethods } from '@/types/spaces'

export type InviteMethod = 'email' | 'link'

/**
 * The method an invitation is made by: the one the inviter prefers if the
 * server allows it, otherwise the one it does allow. Null when neither works,
 * or while the server has not said yet.
 */
export function chooseInviteMethod(methods: InviteMethods | undefined, preferred: InviteMethod): InviteMethod | null {
    if (!methods) return null
    if (methods[preferred]) return preferred
    if (methods.email) return 'email'
    if (methods.link) return 'link'
    return null
}
