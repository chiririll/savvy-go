import { useState } from 'react'
import { MailX } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { SettingsSection } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useClearInvitations } from '@/hooks/use-spaces'

/**
 * An address cannot be invited again for a while after an invitation to it was
 * revoked or ran out, so nobody can be flooded with email. Only a server admin
 * can end that wait early, for one address or for all.
 */
export function ClearInvitationsSection() {
    const { t } = useTranslation('settings')
    const clear = useClearInvitations()
    const [email, setEmail] = useState('')

    return (
        <SettingsSection
            icon={MailX}
            title={t('system.invitations.title')}
            description={t('system.invitations.description')}
        >
            <form
                className="flex flex-col gap-2 sm:flex-row"
                onSubmit={(e) => {
                    e.preventDefault()
                    clear.mutate(email.trim() || undefined, { onSuccess: () => setEmail('') })
                }}
            >
                <Input
                    type="email"
                    value={email}
                    placeholder={t('system.invitations.placeholder')}
                    aria-label={t('system.invitations.placeholder')}
                    onChange={(e) => setEmail(e.target.value)}
                />
                <Button type="submit" variant="outline" disabled={clear.isPending}>
                    {email.trim() ? t('system.invitations.clearOne') : t('system.invitations.clearAll')}
                </Button>
            </form>
        </SettingsSection>
    )
}
