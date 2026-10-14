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
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useInvite, useRevokeInvitation } from '@/hooks/use-spaces'
import { spaceRoleLabel } from '@/lib/labels'
import type { SpaceRole } from '@/types/spaces'

const ROLES: SpaceRole[] = ['admin', 'editor', 'viewer']

interface Sent {
    /** Local list key. */
    key: number
    /** The invitation on the server, which cancelling revokes. */
    id: number
    link: string
    email: string | null
    role: SpaceRole
    emailSent: boolean
}

/**
 * Invites someone to a space by email and/or link. Every invitation made here is
 * added to the list above the form, newest first, each
 * with its own copyable link (without mail the link is all there is).
 */
export function InviteForm({ spaceId }: { spaceId: number }) {
    const { t } = useTranslation('settings')
    const invite = useInvite(spaceId)
    const revoke = useRevokeInvitation(spaceId)
    const [email, setEmail] = useState('')
    const [role, setRole] = useState<SpaceRole>('editor')
    const [sent, setSent] = useState<Sent[]>([])
    const [copied, setCopied] = useState<number | null>(null)
    const [cancelling, setCancelling] = useState<Sent | null>(null)

    const copy = async (item: Sent) => {
        await navigator.clipboard.writeText(item.link)
        setCopied(item.key)
        toast.success(t('spaces.invitations.copied'))
    }

    return (
        <div className="space-y-3">
            {sent.length > 0 && (
                <ul className="max-h-48 divide-y overflow-y-auto rounded-lg border">
                    {sent.map((item) => (
                        <li key={item.key} className="flex items-center justify-between gap-3 px-3 py-2 text-sm">
                            <div className="min-w-0">
                                <p className="truncate">{item.email ?? t('spaces.invitations.anyone')}</p>
                                <p className="truncate text-xs text-muted-foreground">
                                    {spaceRoleLabel(t, item.role)}
                                    {item.emailSent && ` · ${t('spaces.invitations.sent')}`}
                                </p>
                            </div>
                            <Button type="button" variant="outline" size="sm" className="ml-auto" onClick={() => copy(item)}>
                                {copied === item.key ? <Check className="size-4" /> : <Copy className="size-4" />}
                                {t('spaces.invitations.copy')}
                            </Button>
                            <Button
                                type="button"
                                variant="ghost"
                                size="icon"
                                className="-ml-2 shrink-0"
                                aria-label={t('spaces.invitations.cancel')}
                                onClick={() => setCancelling(item)}
                            >
                                <X className="size-4" />
                            </Button>
                        </li>
                    ))}
                </ul>
            )}
            <form
                className="grid gap-2 sm:grid-cols-[1fr_9rem_auto]"
                onSubmit={async (e) => {
                    e.preventDefault()
                    const to = email.trim()
                    const { id, token, emailSent, emailError } = await invite.mutateAsync({ role, email: to || undefined })
                    const link = `${window.location.origin}/invite/${token}`
                    setSent((list) => [{ key: Date.now(), id, link, email: to || null, role, emailSent }, ...list])
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
