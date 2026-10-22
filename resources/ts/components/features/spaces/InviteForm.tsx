import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useInvite } from '@/hooks/use-spaces'
import type { ApiError } from '@/types'
import type { SpaceRole } from '@/types/spaces'
import { InviteFields } from './InviteFields'

interface Props {
    spaceId: number
    /** Called with the new invitation; `link` is only there for a link invitation, and the server does not keep it. */
    onCreated: (created: { id: number; link: string | null }) => void
}

/**
 * Invites someone to a space, at once. The server says how: by email (the link
 * goes to that mailbox only), by a link to copy, or both. The list of
 * invitations is the caller's.
 */
export function InviteForm({ spaceId, onCreated }: Props) {
    const { t } = useTranslation('settings')
    const invite = useInvite(spaceId)

    const add = async ({ email, note, role }: { email: string | null; note: string | null; role: SpaceRole }) => {
        try {
            const { id, token } = await invite.mutateAsync({ role, email: email || undefined, note: note || undefined })
            onCreated({ id, link: token ? `${window.location.origin}/invite/${token}` : null })
            if (email) toast.success(t('spaces.invitations.emailSent', { email }))
        } catch (error) {
            // The server's own reason: already invited, a recent invitation, a member, the mail failing...
            toast.error((error as ApiError).message)
            return false
        }
    }

    return <InviteFields busy={invite.isPending} submitLabel={t('spaces.invitations.add')} onSubmit={add} />
}
