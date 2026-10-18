import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, Copy, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { testId } from '@/lib/test-id'
import { Button } from '@/components/ui/button'
import type { Space } from '@/types/spaces'
import { InvitationRow } from '../InvitationRow'
import { MembersStep } from './MembersStep'
import type { DraftInvitation } from './draft'
import type { InvitationOutcome } from './send-invitations'

interface Props {
    space: Space
    inviting: boolean
    invitations: DraftInvitation[]
    onInvitationsChange: (invitations: DraftInvitation[]) => void
    /** Set once the invitations went out and there is something to show (a link to copy, a refusal). */
    sent: InvitationOutcome[] | null
}

function SentList({ sent }: { sent: InvitationOutcome[] }) {
    const { t } = useTranslation('settings')
    const [copied, setCopied] = useState<string | null>(null)

    const copyLink = async (outcome: InvitationOutcome) => {
        if (!outcome.link) return
        await navigator.clipboard.writeText(outcome.link)
        setCopied(outcome.invitation.key)
        toast.success(t('spaces.invitations.copied'))
    }

    return (
        <div className="space-y-2">
            <p className="text-sm font-medium">{t('spaces.create.done.invitations')}</p>
            <ul className="divide-y rounded-lg border">
                {sent.map((outcome) => (
                    <InvitationRow
                        key={outcome.invitation.key}
                        email={outcome.invitation.email}
                        label={outcome.invitation.note}
                        role={outcome.invitation.role}
                        note={outcome.emailed ? t('spaces.invitations.sent') : undefined}
                        problem={outcome.error ? t('spaces.create.done.notSent', { error: outcome.error }) : undefined}
                    >
                        {outcome.link && (
                            <Button type="button" variant="outline" size="sm" onClick={() => void copyLink(outcome)}>
                                {copied === outcome.invitation.key ? (
                                    <Check className="size-4" />
                                ) : (
                                    <Copy className="size-4" />
                                )}
                                {t('spaces.invitations.copy')}
                            </Button>
                        )}
                    </InvitationRow>
                ))}
            </ul>
            {sent.some((o) => o.error) && (
                <p className="text-xs text-muted-foreground">{t('spaces.create.done.retry')}</p>
            )}
            {sent.some((o) => o.link) && (
                <p className="text-xs text-muted-foreground">{t('spaces.invitations.linkReady')}</p>
            )}
        </div>
    )
}

export function DoneStep({ space, inviting, invitations, onInvitationsChange, sent }: Props) {
    const { t } = useTranslation('settings')

    return (
        <div className="space-y-4">
            <p className="text-sm">{t('spaces.create.done.text', { name: space.name })}</p>
            {sent ? (
                <SentList sent={sent} />
            ) : (
                inviting && <MembersStep invitations={invitations} onChange={onInvitationsChange} />
            )}
        </div>
    )
}

interface FooterProps {
    inviting: boolean
    sent: InvitationOutcome[] | null
    isSending: boolean
    onInvite: () => void
    onLeave: () => void
}

export function DoneFooter({ inviting, sent, isSending, onInvite, onLeave }: FooterProps) {
    const { t } = useTranslation('settings')

    return (
        <>
            {!inviting && !sent && (
                <Button type="button" variant="outline" onClick={onInvite} {...testId('create-space-invite')}>
                    {t('spaces.create.done.invite')}
                </Button>
            )}
            <Button type="button" disabled={isSending} onClick={onLeave} {...testId('create-space-done')}>
                {isSending && <Loader2 className="size-4 animate-spin" />}
                {sent ? t('actions.done', { ns: 'common' }) : t('actions.continue', { ns: 'common' })}
            </Button>
        </>
    )
}
