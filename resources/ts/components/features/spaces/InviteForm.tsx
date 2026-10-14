import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Copy } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useInvite } from '@/hooks/use-spaces'
import { spaceRoleLabel } from '@/lib/labels'
import type { SpaceRole } from '@/types/spaces'

const ROLES: SpaceRole[] = ['admin', 'editor', 'viewer']

/**
 * Invites someone to a space by email and/or link. Without mail the link is
 * all there is, so it is always shown after creating the invitation.
 */
export function InviteForm({ spaceId }: { spaceId: number }) {
    const { t } = useTranslation('settings')
    const invite = useInvite(spaceId)
    const [email, setEmail] = useState('')
    const [role, setRole] = useState<SpaceRole>('editor')
    const [link, setLink] = useState<string | null>(null)

    const copy = async (value: string) => {
        await navigator.clipboard.writeText(value)
        toast.success(t('spaces.invitations.copied'))
    }

    return (
        <div className="space-y-4">
            <form
                className="grid gap-2 sm:grid-cols-[2fr_1fr_auto]"
                onSubmit={async (e) => {
                    e.preventDefault()
                    const to = email.trim()
                    const { token, emailSent, emailError } = await invite.mutateAsync({ role, email: to || undefined })
                    setLink(`${window.location.origin}/invite/${token}`)
                    setEmail('')
                    if (emailSent) toast.success(t('spaces.invitations.emailSent', { email: to }))
                    else if (emailError) toast.error(t('spaces.invitations.emailFailed', { error: emailError }))
                }}
            >
                <Input type="email" placeholder={t('spaces.invitations.emailPlaceholder')} value={email} onChange={(e) => setEmail(e.target.value)} />
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
                <Button type="submit" disabled={invite.isPending}>{t('spaces.invitations.create')}</Button>
            </form>
            {link && (
                <div className="space-y-1.5 rounded-lg border bg-muted/40 p-3">
                    <p className="text-sm">{t('spaces.invitations.linkReady')}</p>
                    <div className="flex gap-2">
                        <Input readOnly value={link} className="font-mono text-xs" />
                        <Button type="button" variant="outline" size="icon" onClick={() => copy(link)}>
                            <Copy className="size-4" />
                        </Button>
                    </div>
                </div>
            )}
        </div>
    )
}
