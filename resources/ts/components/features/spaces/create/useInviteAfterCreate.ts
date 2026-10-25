import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation } from '@tanstack/react-query'
import { toast } from 'sonner'
import { getApiErrorMessage } from '@/lib/api-error'
import type { Space } from '@/types/spaces'
import type { DraftInvitation } from './draft'
import { sendInvitations, type InvitationOutcome } from './send-invitations'

/**
 * The optional invitations on the done screen. They are not part of the draft: they go out when
 * the screen is left, and links and refusals need reading first, so then the screen stays.
 */
export function useInviteAfterCreate(created: Space | null, onDone: () => void) {
    const { t } = useTranslation('settings')
    const [inviting, setInviting] = useState(false)
    const [invitations, setInvitations] = useState<DraftInvitation[]>([])
    // Set once the invitations went out and there is something to show (a link to copy, a refusal).
    const [sent, setSent] = useState<InvitationOutcome[] | null>(null)

    const invite = useMutation({
        mutationFn: () => sendInvitations(created!.id, invitations),
        onSuccess: (outcomes) => (outcomes.some((o) => o.link || o.error) ? setSent(outcomes) : onDone()),
        onError: (error) => toast.error(getApiErrorMessage(error, t('spaces.create.failed'))),
    })

    const reset = () => {
        setInviting(false)
        setInvitations([])
        setSent(null)
    }

    const leave = () => (invitations.length > 0 && !sent ? invite.mutate() : onDone())

    return { inviting, setInviting, invitations, setInvitations, sent, isSending: invite.isPending, leave, reset }
}
