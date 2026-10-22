import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Check, Copy, Mail } from 'lucide-react'
import { SettingsSection } from '@/components/shared'
import { InvitationList } from '@/components/features/spaces/InvitationList'
import { InvitationRow } from '@/components/features/spaces/InvitationRow'
import { InviteForm } from '@/components/features/spaces/InviteForm'
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
import { useRevokeInvitation, useSpaceInvitations } from '@/hooks'
import { formatDateTime } from '@/lib/format'
import { testId } from '@/lib/test-id'
import { Space } from '@/types/spaces'

/**
 * The open invitations of a space: one list, above the form. The server keeps only a
 * hash of a link's token, so a link can be copied from its row until the page is
 * reloaded, and never after.
 */
export function Invitations({ space }: { space: Space }) {
    const { t } = useTranslation('settings')
    const { data: invitations = [] } = useSpaceInvitations(space.id)
    const revoke = useRevokeInvitation(space.id)
    const [links, setLinks] = useState<Record<number, string>>({})
    const [copied, setCopied] = useState<number | null>(null)
    const [cancelling, setCancelling] = useState<(typeof invitations)[number] | null>(null)
    const open = invitations.filter((i) => !i.acceptedAt && (!i.expiresAt || new Date(i.expiresAt) > new Date()))

    const copy = async (id: number, link: string) => {
        await navigator.clipboard.writeText(link)
        setCopied(id)
        toast.success(t('spaces.invitations.copied'))
    }

    return (
        <SettingsSection
            icon={Mail}
            title={t('spaces.invitations.title')}
            description={t('spaces.invitations.description')}
        >
            <div className="space-y-4">
                <InvitationList empty={t('spaces.invitations.none')}>
                    {open.map((i) => {
                        const link = links[i.id]
                        return (
                            <InvitationRow
                                key={i.id}
                                email={i.email}
                                label={i.note}
                                role={i.role}
                                note={
                                    link
                                        ? t('spaces.invitations.linkReady')
                                        : t('spaces.invitations.expires', { date: formatDateTime(i.expiresAt) })
                                }
                            >
                                {link && (
                                    <Button
                                        type="button"
                                        variant="outline"
                                        size="sm"
                                        {...testId('invite-copy')}
                                        onClick={() => void copy(i.id, link)}
                                    >
                                        {copied === i.id ? <Check className="size-4" /> : <Copy className="size-4" />}
                                        {t('spaces.invitations.copy')}
                                    </Button>
                                )}
                                <Button variant="ghost" size="sm" onClick={() => setCancelling(i)}>
                                    {t('spaces.invitations.revoke')}
                                </Button>
                            </InvitationRow>
                        )
                    })}
                </InvitationList>
                <InviteForm
                    spaceId={space.id}
                    onCreated={({ id, link }) => {
                        if (link) setLinks((all) => ({ ...all, [id]: link }))
                    }}
                />
            </div>
            <AlertDialog open={cancelling !== null} onOpenChange={(isOpen) => !isOpen && setCancelling(null)}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>{t('spaces.invitations.cancelTitle')}</AlertDialogTitle>
                        <AlertDialogDescription>
                            {t('spaces.invitations.cancelText', {
                                who: cancelling?.email ?? t('spaces.invitations.anyone'),
                            })}
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>{t('spaces.invitations.keep')}</AlertDialogCancel>
                        <AlertDialogAction onClick={() => cancelling && revoke.mutate(cancelling.id)}>
                            {t('spaces.invitations.cancelConfirm')}
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </SettingsSection>
    )
}
