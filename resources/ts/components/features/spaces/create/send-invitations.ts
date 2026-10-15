import { spacesApi } from '@/api/spaces'
import type { ApiError } from '@/types'
import type { InviteResult, SpaceRole } from '@/types/spaces'
import type { DraftInvitation } from './draft'

/** What came of one entry of the invitation list. */
export interface InvitationOutcome {
    invitation: DraftInvitation
    /** Only a link invitation has one to copy. */
    link: string | null
    emailed: boolean
    /** The server's reason when it was refused, such as an address invited a moment ago. */
    error: string | null
}

type Invite = (spaceId: number, role: SpaceRole, email?: string) => Promise<InviteResult>

/**
 * Sends the draft's invitations one by one once the space exists. One refused
 * does not stop the rest; each outcome says what happened, for the screen that
 * follows.
 */
export async function sendInvitations(
    spaceId: number,
    invitations: DraftInvitation[],
    invite: Invite = spacesApi.invite,
    origin: string = window.location.origin
): Promise<InvitationOutcome[]> {
    const out: InvitationOutcome[] = []
    for (const invitation of invitations) {
        try {
            const { token, emailSent } = await invite(spaceId, invitation.role, invitation.email ?? undefined)
            out.push({ invitation, link: token ? `${origin}/invite/${token}` : null, emailed: emailSent, error: null })
        } catch (error) {
            out.push({ invitation, link: null, emailed: false, error: (error as ApiError).message })
        }
    }
    return out
}
