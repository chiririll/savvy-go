import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useInviteMethods } from '@/hooks/use-spaces'
import { spaceRoleLabel } from '@/lib/labels'
import type { SpaceRole } from '@/types/spaces'
import { chooseInviteMethod, type InviteMethod } from './invite-method'

const ROLES: SpaceRole[] = ['admin', 'editor', 'viewer']

interface Props {
    submitLabel: ReactNode
    busy?: boolean
    /** Shown under the fields; the parent clears it through onEdit. */
    error?: string | null
    onEdit?: () => void
    /**
     * Gets what was entered: an address for an invitation by email, null for a
     * link. Returning false keeps the address in the field, as after a refusal.
     */
    onSubmit: (input: { email: string | null; role: SpaceRole }) => boolean | void | Promise<boolean | void>
}

/**
 * The fields shared by everything that invites someone: which way (the server
 * says what it allows), the address, the role. What happens on submit is the
 * caller's: a settings page sends it at once, the create-space flow only lists it.
 */
export function InviteFields({ submitLabel, busy, error, onEdit, onSubmit }: Props) {
    const { t } = useTranslation('settings')
    const { data: methods } = useInviteMethods()
    const [preferred, setPreferred] = useState<InviteMethod>('email')
    const [email, setEmail] = useState('')
    const [role, setRole] = useState<SpaceRole>('editor')

    const method = chooseInviteMethod(methods, preferred)

    const submit = async (e: React.FormEvent) => {
        e.preventDefault()
        // An empty address is not a link invitation: that is the other method.
        if (method === null || (method === 'email' && !email.trim())) return
        const keep = await onSubmit({ email: method === 'email' ? email.trim() : null, role })
        if (keep !== false) setEmail('')
    }

    return (
        <div className="space-y-3">
            {methods && method === null && <p className="text-sm text-muted-foreground">{t('spaces.invitations.unavailable')}</p>}
            {methods?.email && methods.link && (
                <div className="flex gap-2" role="group" aria-label={t('spaces.invitations.method')}>
                    {(['email', 'link'] as const).map((m) => (
                        <Button
                            key={m}
                            type="button"
                            size="sm"
                            variant={method === m ? 'secondary' : 'outline'}
                            aria-pressed={method === m}
                            onClick={() => setPreferred(m)}
                        >
                            {m === 'email' ? t('spaces.invitations.methodEmail') : t('spaces.invitations.methodLink')}
                        </Button>
                    ))}
                </div>
            )}
            {method !== null && (
                <form
                    className={method === 'email' ? 'grid gap-2 sm:grid-cols-[1fr_9rem_auto]' : 'grid gap-2 sm:grid-cols-[9rem_auto]'}
                    noValidate
                    onSubmit={(e) => void submit(e)}
                >
                    {method === 'email' && (
                        <Input
                            type="email"
                            placeholder={t('spaces.invitations.emailPlaceholder')}
                            value={email}
                            aria-invalid={!!error}
                            onChange={(e) => {
                                setEmail(e.target.value)
                                onEdit?.()
                            }}
                        />
                    )}
                    <Select value={role} onValueChange={(r) => setRole(r as SpaceRole)}>
                        <SelectTrigger className="w-full">
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            {ROLES.map((r) => (
                                <SelectItem key={r} value={r}>{spaceRoleLabel(t, r)}</SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                    <Button type="submit" disabled={busy}>{submitLabel}</Button>
                </form>
            )}
            {error && <p className="text-sm text-destructive">{error}</p>}
            {methods && !methods.email && methods.emailReason && (
                <p className="text-xs text-muted-foreground">{t('spaces.invitations.emailOff', { reason: methods.emailReason })}</p>
            )}
            {methods?.email && !methods.link && <p className="text-xs text-muted-foreground">{t('spaces.invitations.emailRequired')}</p>}
        </div>
    )
}
