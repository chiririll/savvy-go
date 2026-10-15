import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import type { SpaceRole } from '@/types/spaces'
import { InvitationRow } from '../InvitationRow'
import { InviteFields } from '../InviteFields'
import { addInvitation, removeInvitation, type DraftInvitation } from './draft'

interface Props {
    invitations: DraftInvitation[]
    onChange: (invitations: DraftInvitation[]) => void
}

/**
 * Collects who to invite. Nothing is sent from here: the caller keeps the list
 * and sends it when the done screen of the new space is left.
 */
export function MembersStep({ invitations, onChange }: Props) {
    const { t } = useTranslation('settings')
    const [problem, setProblem] = useState<'invalid' | 'duplicate' | null>(null)

    const add = ({ email, note, role }: { email: string | null; note: string | null; role: SpaceRole }) => {
        const next = addInvitation(invitations, { email, note, role })
        if (next === invitations) {
            // addInvitation refuses an address that is not one or is listed already.
            setProblem(invitations.some((i) => i.email?.toLowerCase() === email?.toLowerCase()) ? 'duplicate' : 'invalid')
            return false
        }
        setProblem(null)
        onChange(next)
    }

    return (
        <div className="space-y-4">
            {invitations.length > 0 ? (
                <ul className="max-h-56 divide-y overflow-y-auto rounded-lg border">
                    {invitations.map((item) => (
                        <InvitationRow
                            key={item.key}
                            email={item.email}
                            label={item.note}
                            role={item.role}
                            note={item.email ? t('spaces.create.members.byEmail') : t('spaces.create.members.byLink')}
                        >
                            <Button
                                type="button"
                                variant="ghost"
                                size="icon"
                                aria-label={t('spaces.create.members.remove', { who: item.email ?? t('spaces.invitations.anyone') })}
                                onClick={() => onChange(removeInvitation(invitations, item.key))}
                            >
                                <X className="size-4" />
                            </Button>
                        </InvitationRow>
                    ))}
                </ul>
            ) : (
                <p className="text-sm text-muted-foreground">{t('spaces.create.members.none')}</p>
            )}
            <InviteFields
                submitLabel={t('spaces.create.members.add')}
                error={problem === 'invalid' ? t('spaces.create.members.invalid') : problem === 'duplicate' ? t('spaces.create.members.duplicate') : null}
                onEdit={() => setProblem(null)}
                onSubmit={add}
            />

        </div>
    )
}
