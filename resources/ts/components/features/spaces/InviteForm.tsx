import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Check, Copy, X } from 'lucide-react'
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { useInvite, useRevokeInvitation } from '@/hooks/use-spaces'
import type { ApiError } from '@/types'
import type { SpaceRole } from '@/types/spaces'
import { InvitationRow } from './InvitationRow'
import { InviteFields } from './InviteFields'

interface Sent {
    /** Local list key. */
    key: number
    /** The invitation on the server, which cancelling revokes. */
    id: number
    /** Only a link invitation has one; an emailed one went to the mailbox and nowhere else. */
    link: string | null
    email: string | null
    note: string | null
    role: SpaceRole
}

/**
 * Invites someone to a space, at once. The server says how: by email (the link
 * goes to that mailbox only), by a link to copy, or both. Every invitation made
 * here is added to the list above the form, newest first.
 */
export function InviteForm({ spaceId }: { spaceId: number }) {
    const { t } = useTranslation('settings')
    const invite = useInvite(spaceId)
    const revoke = useRevokeInvitation(spaceId)
    const [sent, setSent] = useState<Sent[]>([])
    const [copied, setCopied] = useState<number | null>(null)
    const [cancelling, setCancelling] = useState<Sent | null>(null)

    const copy = async (item: Sent) => {
        if (!item.link) return
        await navigator.clipboard.writeText(item.link)
        setCopied(item.key)
        toast.success(t('spaces.invitations.copied'))
    }

    const add = async ({ email, note, role }: { email: string | null; note: string | null; role: SpaceRole }) => {
        try {
            const { id, token } = await invite.mutateAsync({ role, email: email || undefined, note: note || undefined })
            const link = token ? `${window.location.origin}/invite/${token}` : null
            setSent((list) => [{ key: Date.now(), id, link, email: email || null, note, role }, ...list])
            if (email) toast.success(t('spaces.invitations.emailSent', { email }))
        } catch (error) {
            // The server's own reason: already invited, a recent invitation, a member, the mail failing...
            toast.error((error as ApiError).message)
            return false
        }
    }

    return (
        <div className="space-y-3">
            {sent.length > 0 && (
                <ul className="max-h-48 divide-y overflow-y-auto rounded-lg border">
                    {sent.map((item) => (
                        <InvitationRow key={item.key} email={item.email} label={item.note} role={item.role} note={item.email ? t('spaces.invitations.sent') : undefined}>
                            {item.link && (
                                <Button type="button" variant="outline" size="sm" onClick={() => void copy(item)}>
                                    {copied === item.key ? <Check className="size-4" /> : <Copy className="size-4" />}
                                    {t('spaces.invitations.copy')}
                                </Button>
                            )}
                            <Button type="button" variant="ghost" size="icon" aria-label={t('spaces.invitations.cancel')} onClick={() => setCancelling(item)}>
                                <X className="size-4" />
                            </Button>
                        </InvitationRow>
                    ))}
                </ul>
            )}
            <InviteFields busy={invite.isPending} submitLabel={t('spaces.invitations.add')} onSubmit={add} />
            <AlertDialog open={cancelling !== null} onOpenChange={(open) => !open && setCancelling(null)}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>{t('spaces.invitations.cancelTitle')}</AlertDialogTitle>
                        <AlertDialogDescription>
                            {t('spaces.invitations.cancelText', { who: cancelling?.email ?? t('spaces.invitations.anyone') })}
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>{t('spaces.invitations.keep')}</AlertDialogCancel>
                        <AlertDialogAction
                            onClick={() => {
                                const item = cancelling
                                if (!item) return
                                revoke.mutate(item.id, { onSuccess: () => setSent((list) => list.filter((i) => i.key !== item.key)) })
                            }}
                        >
                            {t('spaces.invitations.cancelConfirm')}
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    )
}
