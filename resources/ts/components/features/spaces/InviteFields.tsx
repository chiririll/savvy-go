import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { EmailLanguageSelect } from '@/components/shared/appearance/EmailLanguageSelect'
import { useInviteMethods } from '@/hooks/use-spaces'
import { toAppLocale, type AppLocale } from '@/lib/i18n'
import { spaceRoleLabel } from '@/lib/labels'
import { testId } from '@/lib/test-id'
import type { SpaceRole } from '@/types/spaces'
import { chooseInviteMethod, type InviteMethod } from './invite-method'

const ROLES: SpaceRole[] = ['admin', 'editor', 'viewer']

interface Props {
    /** The button is only a plus, so this is what a screen reader hears. */
    submitLabel: string
    busy?: boolean
    /** Shown under the fields; the parent clears it through onEdit. */
    error?: string | null
    onEdit?: () => void
    /**
     * Gets what was entered: an address for an invitation by email, or null and
     * the optional note for a link. `locale` is the language of the email, null for a link.
     * Returning false keeps the address in the field, as after a refusal.
     */
    onSubmit: (input: {
        email: string | null
        note: string | null
        role: SpaceRole
        locale: AppLocale | null
    }) => boolean | void | Promise<boolean | void>
}

/**
 * The fields shared by everything that invites someone: which way (the server
 * says what it allows), the address, the role. What happens on submit is the
 * caller's: a settings page sends it at once, the create-space flow only lists it.
 */
export function InviteFields({ submitLabel, busy, error, onEdit, onSubmit }: Props) {
    const { t, i18n } = useTranslation('settings')
    const { data: methods } = useInviteMethods()
    const [preferred, setPreferred] = useState<InviteMethod>('email')
    const [email, setEmail] = useState('')
    const [note, setNote] = useState('')
    const [role, setRole] = useState<SpaceRole>('editor')
    // Not kept between invitations: each email starts from the language of the interface.
    const [chosen, setChosen] = useState<AppLocale | null>(null)
    const locale = chosen ?? toAppLocale(i18n.resolvedLanguage ?? i18n.language)

    const method = chooseInviteMethod(methods, preferred)

    const submit = async (e: React.FormEvent) => {
        e.preventDefault()
        // An empty address is not a link invitation: that is the other method.
        if (method === null || (method === 'email' && !email.trim())) return
        const keep = await onSubmit({
            email: method === 'email' ? email.trim() : null,
            note: method === 'link' ? note.trim() || null : null,
            role,
            locale: method === 'email' ? locale : null,
        })
        if (keep !== false) {
            setEmail('')
            setNote('')
        }
    }

    return (
        <div className="space-y-3">
            {methods && method === null && (
                <p className="text-sm text-muted-foreground">{t('spaces.invitations.unavailable')}</p>
            )}
            {methods?.email && methods.link && (
                <div className="flex gap-2" role="group" aria-label={t('spaces.invitations.method')}>
                    {(['email', 'link'] as const).map((m) => (
                        <Button
                            key={m}
                            type="button"
                            size="sm"
                            variant={method === m ? 'secondary' : 'outline'}
                            aria-pressed={method === m}
                            {...testId(`invite-method-${m}`)}
                            onClick={() => setPreferred(m)}
                        >
                            {m === 'email' ? t('spaces.invitations.methodEmail') : t('spaces.invitations.methodLink')}
                        </Button>
                    ))}
                </div>
            )}
            {method !== null && (
                <form
                    className="grid grid-cols-[1fr_2.25rem] gap-2 sm:grid-cols-[1fr_9rem_2.25rem]"
                    noValidate
                    onSubmit={(e) => void submit(e)}
                >
                    {method === 'email' ? (
                        <Input
                            className="col-span-2 sm:col-span-1"
                            type="email"
                            placeholder={t('spaces.invitations.emailPlaceholder')}
                            aria-label={t('spaces.invitations.emailPlaceholder')}
                            value={email}
                            aria-invalid={!!error}
                            {...testId('invite-email')}
                            onChange={(e) => {
                                setEmail(e.target.value)
                                onEdit?.()
                            }}
                        />
                    ) : (
                        <Input
                            className="col-span-2 sm:col-span-1"
                            placeholder={t('spaces.invitations.notePlaceholder')}
                            aria-label={t('spaces.invitations.notePlaceholder')}
                            value={note}
                            maxLength={100}
                            {...testId('invite-note')}
                            onChange={(e) => setNote(e.target.value)}
                        />
                    )}
                    <Select value={role} onValueChange={(r) => setRole(r as SpaceRole)}>
                        <SelectTrigger className="w-full" aria-label={t('common:fields.role')}>
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            {ROLES.map((r) => (
                                <SelectItem key={r} value={r}>
                                    {spaceRoleLabel(t, r)}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                    <Button
                        type="submit"
                        size="icon"
                        disabled={busy}
                        aria-label={submitLabel}
                        title={submitLabel}
                        {...testId('invite-add')}
                    >
                        <Plus className="size-4" />
                    </Button>
                    {method === 'email' && (
                        <div className="col-span-full flex items-center gap-2" {...testId('invite-locale')}>
                            <span className="text-sm text-muted-foreground">{t('common:language.email')}</span>
                            <EmailLanguageSelect className="w-36" value={locale} onChange={setChosen} />
                        </div>
                    )}
                </form>
            )}
            {error && <p className="text-sm text-destructive">{error}</p>}
            {methods && !methods.email && methods.emailReason && (
                <p className="text-xs text-muted-foreground">
                    {t('spaces.invitations.emailOff', { reason: methods.emailReason })}
                </p>
            )}
            {methods?.email && !methods.link && (
                <p className="text-xs text-muted-foreground">{t('spaces.invitations.emailRequired')}</p>
            )}
        </div>
    )
}
