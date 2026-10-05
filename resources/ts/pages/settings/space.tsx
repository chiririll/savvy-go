import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { AlertTriangle, ArrowLeftRight, Boxes, Copy, History, Link2, LogOut, Mail, Trash2, Unlink, Users } from 'lucide-react'
import { Page, PageHeader, SettingsSection } from '@/components/shared'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import {
    useAccounts,
    useCurrentSpace,
    useDeleteSpace,
    useDeleteTransfer,
    useInvite,
    useLeaveSpace,
    useLinkSpace,
    useRemoveMember,
    useRenameSpace,
    useResolveTransfer,
    useRevokeInvitation,
    useSetMemberRole,
    useSpaceAudit,
    useSpaceDetails,
    useSpaceInvitations,
    useSpaceLinks,
    useSpaceMembers,
    useSpaces,
    useSpaceTransfers,
    useUnlinkSpace,
    useUpdateTransfer,
} from '@/hooks'
import { formatBytes, formatDateTime } from '@/lib/format'
import { useUser } from '@/stores/auth'
import { Space, SpaceRole, SpaceTransfer } from '@/types/spaces'
import { auditActionLabel, spaceRoleLabel, transferReviewLabel } from '@/lib/labels'

const ROLES: SpaceRole[] = ['admin', 'editor', 'viewer']

function Confirm({
    trigger,
    title,
    description,
    action,
    onConfirm,
}: {
    trigger: React.ReactNode
    title: string
    description: string
    action: string
    onConfirm: () => void
}) {
    const { t } = useTranslation('common')
    return (
        <AlertDialog>
            <AlertDialogTrigger asChild>{trigger}</AlertDialogTrigger>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>{title}</AlertDialogTitle>
                    <AlertDialogDescription>{description}</AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                    <AlertDialogCancel>{t('actions.cancel')}</AlertDialogCancel>
                    <AlertDialogAction onClick={onConfirm}>{action}</AlertDialogAction>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    )
}

function General({ space, isAdmin }: { space: Space; isAdmin: boolean }) {
    const { t } = useTranslation('settings')
    const navigate = useNavigate()
    const { data: details } = useSpaceDetails(space.id)
    const [name, setName] = useState(space.name)
    const rename = useRenameSpace(space.id)
    const remove = useDeleteSpace(space.id)
    const leave = useLeaveSpace(space.id)

    return (
        <SettingsSection icon={Boxes} title={t('spaces.general.title')} description={t('spaces.general.description')}>
            <div className="space-y-4">
                <div className="space-y-1.5">
                    <Label htmlFor="space-name">{t('spaces.fields.name')}</Label>
                    <div className="flex gap-2">
                        <Input id="space-name" value={name} disabled={!isAdmin} maxLength={100} onChange={(e) => setName(e.target.value)} />
                        {isAdmin && name.trim() !== space.name && (
                            <Button disabled={!name.trim() || rename.isPending} onClick={() => rename.mutate(name.trim())}>
                                {t('actions.save', { ns: 'common' })}
                            </Button>
                        )}
                    </div>
                </div>
                {details && (
                    <p className="text-sm text-muted-foreground">
                        {details.quota > 0
                            ? t('spaces.general.usageQuota', { size: formatBytes(details.size), quota: formatBytes(details.quota) })
                            : t('spaces.general.usage', { size: formatBytes(details.size) })}
                    </p>
                )}
                <div className="flex flex-wrap gap-2">
                    <Confirm
                        trigger={
                            <Button variant="outline" size="sm">
                                <LogOut className="mr-2 size-4" />
                                {t('spaces.general.leave')}
                            </Button>
                        }
                        title={t('spaces.general.leaveTitle', { name: space.name })}
                        description={t('spaces.general.leaveDescription')}
                        action={t('spaces.general.leave')}
                        onConfirm={() => leave.mutate(undefined, { onSuccess: () => navigate('/') })}
                    />
                    {isAdmin && (
                        <Confirm
                            trigger={
                                <Button variant="destructive" size="sm">
                                    <Trash2 className="mr-2 size-4" />
                                    {t('spaces.general.delete')}
                                </Button>
                            }
                            title={t('spaces.general.deleteTitle', { name: space.name })}
                            description={t('spaces.general.deleteDescription')}
                            action={t('spaces.general.delete')}
                            onConfirm={() => remove.mutate(undefined, { onSuccess: () => navigate('/') })}
                        />
                    )}
                </div>
            </div>
        </SettingsSection>
    )
}

function Members({ space, isAdmin }: { space: Space; isAdmin: boolean }) {
    const { t } = useTranslation('settings')
    const me = useUser()
    const { data: members = [] } = useSpaceMembers(space.id)
    const setRole = useSetMemberRole(space.id)
    const remove = useRemoveMember(space.id)

    return (
        <SettingsSection icon={Users} title={t('spaces.members.title')} description={t('spaces.members.description')}>
            <ul className="divide-y rounded-lg border">
                {members.map((m) => (
                    <li key={m.userId} className="flex items-center justify-between gap-3 px-3 py-2">
                        <div className="min-w-0">
                            <p className="truncate text-sm font-medium">
                                {m.name} {m.userId === me?.id && <span className="text-muted-foreground">({t('spaces.members.you')})</span>}
                            </p>
                            <p className="truncate text-xs text-muted-foreground">{m.email}</p>
                        </div>
                        <div className="flex items-center gap-2">
                            {isAdmin ? (
                                <Select value={m.role} onValueChange={(role) => setRole.mutate({ userId: m.userId, role: role as SpaceRole })}>
                                    <SelectTrigger className="h-8 w-32">
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        {ROLES.map((r) => (
                                            <SelectItem key={r} value={r}>{spaceRoleLabel(t, r)}</SelectItem>
                                        ))}
                                    </SelectContent>
                                </Select>
                            ) : (
                                <Badge variant="secondary">{spaceRoleLabel(t, m.role)}</Badge>
                            )}
                            {isAdmin && m.userId !== me?.id && (
                                <Button variant="ghost" size="icon" title={t('spaces.members.remove')} onClick={() => remove.mutate(m.userId)}>
                                    <Trash2 className="size-4" />
                                </Button>
                            )}
                        </div>
                    </li>
                ))}
            </ul>
        </SettingsSection>
    )
}

function Invitations({ space }: { space: Space }) {
    const { t } = useTranslation('settings')
    const { data: invitations = [] } = useSpaceInvitations(space.id)
    const invite = useInvite(space.id)
    const revoke = useRevokeInvitation(space.id)
    const [email, setEmail] = useState('')
    const [role, setRole] = useState<SpaceRole>('editor')
    const [link, setLink] = useState<string | null>(null)
    const open = invitations.filter((i) => !i.acceptedAt && (!i.expiresAt || new Date(i.expiresAt) > new Date()))

    const copy = async (value: string) => {
        await navigator.clipboard.writeText(value)
        toast.success(t('spaces.invitations.copied'))
    }

    return (
        <SettingsSection icon={Mail} title={t('spaces.invitations.title')} description={t('spaces.invitations.description')}>
            <div className="space-y-4">
                <form
                    className="grid gap-2 sm:grid-cols-[2fr_1fr_auto]"
                    onSubmit={async (e) => {
                        e.preventDefault()
                        const { token } = await invite.mutateAsync({ role, email: email.trim() || undefined })
                        setLink(`${window.location.origin}/invite/${token}`)
                        setEmail('')
                    }}
                >
                    <Input type="email" placeholder={t('spaces.invitations.emailPlaceholder')} value={email} onChange={(e) => setEmail(e.target.value)} />
                    <Select value={role} onValueChange={(r) => setRole(r as SpaceRole)}>
                        <SelectTrigger>
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
                            <Button variant="outline" size="icon" onClick={() => copy(link)}>
                                <Copy className="size-4" />
                            </Button>
                        </div>
                    </div>
                )}
                {open.length > 0 && (
                    <ul className="divide-y rounded-lg border">
                        {open.map((i) => (
                            <li key={i.id} className="flex items-center justify-between gap-3 px-3 py-2 text-sm">
                                <div className="min-w-0">
                                    <p className="truncate">{i.email ?? t('spaces.invitations.anyone')} · {spaceRoleLabel(t, i.role)}</p>
                                    <p className="truncate text-xs text-muted-foreground">
                                        {t('spaces.invitations.expires', { date: formatDateTime(i.expiresAt) })}
                                    </p>
                                </div>
                                <Button variant="ghost" size="sm" onClick={() => revoke.mutate(i.id)}>
                                    {t('spaces.invitations.revoke')}
                                </Button>
                            </li>
                        ))}
                    </ul>
                )}
            </div>
        </SettingsSection>
    )
}

function Links({ space, isAdmin }: { space: Space; isAdmin: boolean }) {
    const { t } = useTranslation('settings')
    const { data: links = [] } = useSpaceLinks(space.id)
    const { data: mine = [] } = useSpaces()
    const link = useLinkSpace(space.id)
    const unlink = useUnlinkSpace(space.id)
    const [target, setTarget] = useState('')
    const candidates = mine.filter((s) => s.role === 'admin' && s.id !== space.id && !links.some((l) => l.id === s.id))

    return (
        <SettingsSection icon={Link2} title={t('spaces.links.title')} description={t('spaces.links.description')}>
            <div className="space-y-4">
                {links.length ? (
                    <ul className="divide-y rounded-lg border">
                        {links.map((l) => (
                            <li key={l.id} className="flex items-center justify-between gap-3 px-3 py-2 text-sm">
                                <span className="truncate">{l.name}</span>
                                {isAdmin && (
                                    <Confirm
                                        trigger={
                                            <Button variant="ghost" size="sm">
                                                <Unlink className="mr-2 size-4" />
                                                {t('spaces.links.unlink')}
                                            </Button>
                                        }
                                        title={t('spaces.links.unlinkTitle', { name: l.name })}
                                        description={t('spaces.links.unlinkDescription')}
                                        action={t('spaces.links.unlink')}
                                        onConfirm={() => unlink.mutate(l.id)}
                                    />
                                )}
                            </li>
                        ))}
                    </ul>
                ) : (
                    <p className="text-sm text-muted-foreground">{t('spaces.links.none')}</p>
                )}
                {isAdmin && (
                    candidates.length ? (
                        <div className="flex gap-2">
                            <Select value={target} onValueChange={setTarget}>
                                <SelectTrigger className="flex-1">
                                    <SelectValue placeholder={t('spaces.links.choose')} />
                                </SelectTrigger>
                                <SelectContent>
                                    {candidates.map((c) => (
                                        <SelectItem key={c.id} value={String(c.id)}>{c.name}</SelectItem>
                                    ))}
                                </SelectContent>
                            </Select>
                            <Button disabled={!target || link.isPending} onClick={() => link.mutate(Number(target), { onSuccess: () => setTarget('') })}>
                                {t('spaces.links.link')}
                            </Button>
                        </div>
                    ) : (
                        <p className="text-xs text-muted-foreground">{t('spaces.links.howTo')}</p>
                    )
                )}
            </div>
        </SettingsSection>
    )
}

/** Transfers that need someone: pending after a restore, missing an account, unverifiable or frozen. */
function TransfersNeedingAttention({ space }: { space: Space }) {
    const { t } = useTranslation('settings')
    const { data: transfers = [] } = useSpaceTransfers(space.id)
    const { data: accounts = [] } = useAccounts()
    const resolve = useResolveTransfer(space.id)
    const update = useUpdateTransfer(space.id)
    const remove = useDeleteTransfer(space.id)
    const [choice, setChoice] = useState<Record<string, string>>({})
    const items = transfers.filter((tr) => tr.status !== 'ok' || !tr.verified || tr.frozen)

    if (!items.length) return null

    const amount = (tr: SpaceTransfer) => {
        const side = tr.outgoing ? tr.from : tr.to
        return `${tr.outgoing ? '−' : '+'}${side.amount} ${side.currency}`
    }

    const setAccount = (tr: SpaceTransfer) => {
        const id = Number(choice[tr.uuid])
        update.mutate({ uuid: tr.uuid, data: tr.outgoing ? { from_account_id: id } : { to_account_id: id } })
    }

    return (
        <SettingsSection icon={AlertTriangle} title={t('spaces.transfers.title')} description={t('spaces.transfers.description')}>
            <ul className="divide-y rounded-lg border">
                {items.map((tr) => (
                    <li key={tr.uuid} className="space-y-2 px-3 py-3 text-sm">
                        <div className="flex flex-wrap items-center justify-between gap-2">
                            <div className="flex min-w-0 items-center gap-2">
                                <ArrowLeftRight className="size-4 shrink-0 text-muted-foreground" />
                                <span className="truncate">
                                    {tr.description || t('spaces.transfers.untitled')} · {tr.otherSpace?.name ?? t('spaces.transfers.unknownSpace')} · {tr.date}
                                </span>
                            </div>
                            <span className="font-medium tabular-nums">{amount(tr)}</span>
                        </div>
                        <div className="flex flex-wrap gap-1.5">
                            {tr.review && <Badge variant="outline">{transferReviewLabel(t, tr.review)}</Badge>}
                            {tr.status === 'needs_attention' && <Badge variant="outline">{t('spaces.transfers.needsAccount')}</Badge>}
                            {!tr.verified && <Badge variant="destructive">{t('spaces.transfers.unverified')}</Badge>}
                            {tr.frozen && <Badge variant="secondary">{t('spaces.transfers.frozen')}</Badge>}
                        </div>
                        {tr.remote && (
                            <p className="text-xs text-muted-foreground">
                                {tr.remote.deleted
                                    ? t('spaces.transfers.remoteDeleted')
                                    : t('spaces.transfers.remoteVersion', {
                                          amount: `${(tr.outgoing ? tr.remote.from : tr.remote.to).amount} ${(tr.outgoing ? tr.remote.from : tr.remote.to).currency}`,
                                          date: tr.remote.date,
                                      })}
                            </p>
                        )}
                        <div className="flex flex-wrap gap-2">
                            {tr.status === 'pending' && (
                                <>
                                    <Button size="sm" onClick={() => resolve.mutate({ uuid: tr.uuid, decision: 'accept' })}>
                                        {t('spaces.transfers.accept')}
                                    </Button>
                                    <Button size="sm" variant="outline" disabled={!tr.verified && tr.review !== 'created_remote'}
                                        onClick={() => resolve.mutate({ uuid: tr.uuid, decision: 'reject' })}>
                                        {t('spaces.transfers.reject')}
                                    </Button>
                                </>
                            )}
                            {!tr.verified && (
                                <Button size="sm" variant="outline" onClick={() => resolve.mutate({ uuid: tr.uuid, decision: 'trust' })}>
                                    {t('spaces.transfers.trust')}
                                </Button>
                            )}
                            {tr.status === 'needs_attention' && (
                                <>
                                    <Select value={choice[tr.uuid] ?? ''} onValueChange={(v) => setChoice({ ...choice, [tr.uuid]: v })}>
                                        <SelectTrigger className="h-8 w-48">
                                            <SelectValue placeholder={t('spaces.transfers.chooseAccount')} />
                                        </SelectTrigger>
                                        <SelectContent>
                                            {accounts.map((a) => (
                                                <SelectItem key={a.id} value={String(a.id)}>{a.name}</SelectItem>
                                            ))}
                                        </SelectContent>
                                    </Select>
                                    <Button size="sm" disabled={!choice[tr.uuid]} onClick={() => setAccount(tr)}>
                                        {t('actions.save', { ns: 'common' })}
                                    </Button>
                                </>
                            )}
                            {tr.status !== 'pending' && !tr.frozen && tr.verified && (
                                <Button size="sm" variant="ghost" onClick={() => remove.mutate(tr.uuid)}>
                                    {t('actions.delete', { ns: 'common' })}
                                </Button>
                            )}
                        </div>
                    </li>
                ))}
            </ul>
        </SettingsSection>
    )
}

function Audit({ space }: { space: Space }) {
    const { t } = useTranslation('settings')
    const { data: entries = [] } = useSpaceAudit(space.id)
    return (
        <SettingsSection icon={History} title={t('spaces.audit.title')} description={t('spaces.audit.description')}>
            {entries.length ? (
                <ul className="divide-y rounded-lg border text-sm">
                    {entries.map((e) => (
                        <li key={e.id} className="flex justify-between gap-3 px-3 py-2">
                            <span>
                                {e.actor || t('spaces.audit.someone')} — {auditActionLabel(t, e.action)}
                                {e.details ? ` (${e.details})` : ''}
                            </span>
                            <span className="shrink-0 text-xs text-muted-foreground">{formatDateTime(e.createdAt)}</span>
                        </li>
                    ))}
                </ul>
            ) : (
                <p className="text-sm text-muted-foreground">{t('spaces.audit.none')}</p>
            )}
        </SettingsSection>
    )
}

export default function SpaceSettingsPage() {
    const { t } = useTranslation('settings')
    const space = useCurrentSpace()
    if (!space) return <></>
    const isAdmin = space.role === 'admin'

    return (
        <Page title={t('spaces.title')}>
            <PageHeader title={space.name} description={t('spaces.description', { role: spaceRoleLabel(t, space.role) })} />
            <div className="grid items-start gap-6 lg:grid-cols-2">
                <div className="space-y-6">
                    <General key={space.id} space={space} isAdmin={isAdmin} />
                    <Members space={space} isAdmin={isAdmin} />
                    {isAdmin && <Invitations space={space} />}
                </div>
                <div className="space-y-6">
                    <TransfersNeedingAttention space={space} />
                    <Links space={space} isAdmin={isAdmin} />
                    {isAdmin && <Audit space={space} />}
                </div>
            </div>
        </Page>
    )
}
