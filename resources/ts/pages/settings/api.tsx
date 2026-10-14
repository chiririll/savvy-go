import { testId } from '@/lib/test-id'
import { UserSettingsTabs } from '@/components/features/spaces/UserSettingsTabs'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Check, Copy, ExternalLink, KeyRound, Plus, Trash2 } from 'lucide-react'
import { Page, PageHeader, FormWrapper, ResponsiveDialog } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
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
import { useApiTokens, useCreateApiToken, useRevokeApiToken, useSpaces } from '@/hooks'
import { PermissionGrid } from '@/components/features/api-tokens/PermissionGrid'
import { SpacePicker, hasSpaces, type SpaceChoice } from '@/components/features/api-tokens/SpacePicker'
import { intlLocale } from '@/lib/i18n'
import { API_RESOURCES, permissionLevel, type ApiPermission, type ApiResource, type ApiTokenSummary } from '@/types'
import { apiResourceLabel } from '@/lib/labels'

type Expiry = 'never' | '30' | '90' | '365'

/** What a token holds, as two lists of resources: the ones it only reads and the ones it also changes. */
function grantedResources(token: ApiTokenSummary): { read: ApiResource[]; write: ApiResource[] } {
    const read: ApiResource[] = []
    const write: ApiResource[] = []
    for (const resource of API_RESOURCES) {
        const level = permissionLevel(token.permissions, resource)
        if (level === 'write') write.push(resource)
        else if (level === 'read') read.push(resource)
    }
    return { read, write }
}

/** What a token may do, in words. */
function TokenGrants({ token }: { token: ApiTokenSummary }) {
    const { t } = useTranslation('settings')
    const { read, write } = grantedResources(token)
    const list = (items: ApiResource[]) => items.map((r) => apiResourceLabel(t, r)).join(', ')
    if (read.length === 0 && write.length === 0) {
        return <p className="text-xs text-muted-foreground">{t('api.tokens.noAccess')}</p>
    }
    return (
        <div className="space-y-0.5 text-xs text-muted-foreground">
            {read.length > 0 && <p>{t('api.tokens.canRead', { list: list(read) })}</p>}
            {write.length > 0 && <p>{t('api.tokens.canWrite', { list: list(write) })}</p>}
        </div>
    )
}

function expiryToIso(expiry: Expiry): string | null {
    if (expiry === 'never') return null
    const date = new Date()
    date.setDate(date.getDate() + Number(expiry))
    return date.toISOString()
}

export default function ApiSettingsPage() {
    const { t } = useTranslation('settings')
    const { t: tCommon } = useTranslation('common')

    const { data: tokens, isLoading } = useApiTokens()
    const createToken = useCreateApiToken()
    const revokeToken = useRevokeApiToken()
    const { data: spaces = [] } = useSpaces()

    const [showCreate, setShowCreate] = useState(false)
    const [name, setName] = useState('')
    const [permissions, setPermissions] = useState<ApiPermission[]>([])
    const [spaceChoice, setSpaceChoice] = useState<SpaceChoice>({ all: false, ids: [] })
    const [expiry, setExpiry] = useState<Expiry>('never')
    const [createdToken, setCreatedToken] = useState<string | null>(null)
    const [copied, setCopied] = useState(false)

    const formatDate = (value: string) => new Date(value).toLocaleDateString(intlLocale())

    const resetForm = () => {
        setName('')
        setPermissions([])
        setSpaceChoice({ all: false, ids: [] })
        setExpiry('never')
    }

    const handleCreate = async () => {
        try {
            const created = await createToken.mutateAsync({
                name: name.trim(),
                permissions,
                all_spaces: spaceChoice.all,
                space_ids: spaceChoice.ids,
                expires_at: expiryToIso(expiry),
            })
            setShowCreate(false)
            resetForm()
            setCreatedToken(created.token)
            setCopied(false)
        } catch {
            // toasted by the mutation
        }
    }

    const handleCopy = async () => {
        if (!createdToken) return
        try {
            await navigator.clipboard.writeText(createdToken)
            setCopied(true)
        } catch {
            toast.error(t('api.copyFailed'))
        }
    }

    return (
        <Page title={t('api.title')}>
            <PageHeader title={t('api.heading')} description={t('api.description')} />
            <UserSettingsTabs />

            <FormWrapper>
                <section className="rounded-xl border bg-card shadow-sm">
                    <header className="flex flex-wrap items-start justify-between gap-3 border-b px-5 py-4">
                        <div className="flex items-start gap-3">
                            <div className="flex size-9 shrink-0 items-center justify-center rounded-lg border bg-muted/50 text-muted-foreground">
                                <KeyRound className="size-4" />
                            </div>
                            <div className="space-y-0.5">
                                <h3 className="text-sm font-semibold leading-none">{t('api.tokens.title')}</h3>
                                <p className="text-xs text-muted-foreground">{t('api.tokens.description')}</p>
                            </div>
                        </div>
                        <div className="flex shrink-0 gap-2">
                            <Button variant="outline" size="sm" asChild>
                                <a href="/api/docs" target="_blank" rel="noreferrer">
                                    <ExternalLink className="size-4" />
                                    {t('api.docs')}
                                </a>
                            </Button>
                            <Button size="sm" onClick={() => setShowCreate(true)} {...testId('create-token')}>
                                <Plus className="size-4" />
                                {t('api.tokens.add')}
                            </Button>
                        </div>
                    </header>

                    <div className="p-5">
                        {isLoading ? (
                            <Skeleton className="h-16 w-full" />
                        ) : tokens && tokens.length > 0 ? (
                            <div className="divide-y rounded-lg border">
                                {tokens.map((token) => (
                                    <div key={token.id} className="flex items-center justify-between gap-4 px-4 py-3.5">
                                        <div className="min-w-0 space-y-1">
                                            <div className="flex flex-wrap items-center gap-2">
                                                <span className="truncate text-sm font-medium">{token.name}</span>
                                                <Badge variant="secondary">
                                                    {token.all_spaces
                                                        ? t('api.tokens.allSpaces')
                                                        : t('api.tokens.spaceCount', { count: token.space_ids.length })}
                                                </Badge>
                                                <code className="text-xs text-muted-foreground">{token.prefix}…</code>
                                            </div>
                                            <TokenGrants token={token} />
                                            <p className="text-xs text-muted-foreground">
                                                {token.last_used_at
                                                    ? t('api.tokens.lastUsed', { date: formatDate(token.last_used_at) })
                                                    : t('api.tokens.neverUsed')}
                                                {' · '}
                                                {token.expires_at
                                                    ? t('api.tokens.expires', { date: formatDate(token.expires_at) })
                                                    : t('api.tokens.noExpiry')}
                                            </p>
                                        </div>
                                        <AlertDialog>
                                            <AlertDialogTrigger asChild>
                                                <Button
                                                    variant="ghost"
                                                    size="icon"
                                                    disabled={revokeToken.isPending}
                                                    aria-label={t('api.tokens.revokeAria')}
                                                >
                                                    <Trash2 className="size-4" />
                                                </Button>
                                            </AlertDialogTrigger>
                                            <AlertDialogContent>
                                                <AlertDialogHeader>
                                                    <AlertDialogTitle>{t('api.tokens.revokeTitle')}</AlertDialogTitle>
                                                    <AlertDialogDescription>
                                                        {t('api.tokens.revokeDescription', { name: token.name })}
                                                    </AlertDialogDescription>
                                                </AlertDialogHeader>
                                                <AlertDialogFooter>
                                                    <AlertDialogCancel>{tCommon('actions.cancel')}</AlertDialogCancel>
                                                    <AlertDialogAction onClick={() => revokeToken.mutate(token.id)}>
                                                        {t('api.tokens.revoke')}
                                                    </AlertDialogAction>
                                                </AlertDialogFooter>
                                            </AlertDialogContent>
                                        </AlertDialog>
                                    </div>
                                ))}
                            </div>
                        ) : (
                            <p className="rounded-lg border border-dashed px-4 py-6 text-center text-sm text-muted-foreground">
                                {t('api.tokens.empty')}
                            </p>
                        )}
                    </div>
                </section>
            </FormWrapper>

            <ResponsiveDialog
                open={showCreate}
                onOpenChange={(open) => {
                    setShowCreate(open)
                    if (!open) resetForm()
                }}
                title={t('api.create.title')}
                description={t('api.create.description')}
                footer={
                    <>
                        <Button variant="outline" onClick={() => setShowCreate(false)}>
                            {tCommon('actions.cancel')}
                        </Button>
                        <Button
                            onClick={handleCreate}
                            disabled={!name.trim() || !hasSpaces(spaceChoice) || createToken.isPending}
                        >
                            {tCommon('actions.create')}
                        </Button>
                    </>
                }
            >
                <div className="space-y-4">
                    <div className="space-y-2">
                        <Label htmlFor="api-token-name">{t('api.create.name')}</Label>
                        <Input
                            id="api-token-name"
                            placeholder={t('api.create.namePlaceholder')}
                            value={name}
                            maxLength={100}
                            autoFocus
                            onChange={(e) => setName(e.target.value)}
                        />
                    </div>
                    <div className="space-y-2">
                        <Label>{t('api.create.spaces')}</Label>
                        <SpacePicker idPrefix="token-space" spaces={spaces} value={spaceChoice} onChange={setSpaceChoice} />
                        <p className="text-xs text-muted-foreground">
                            {hasSpaces(spaceChoice) ? t('api.create.spacesHint') : t('api.spaces.none')}
                        </p>
                    </div>
                    <div className="space-y-2">
                        <Label>{t('api.create.permissions')}</Label>
                        <PermissionGrid value={permissions} onChange={setPermissions} />
                        <p className="text-xs text-muted-foreground">{t('api.create.permissionsHint')}</p>
                    </div>
                    <div className="space-y-2">
                        <Label>{t('api.create.expiry')}</Label>
                        <Select value={expiry} onValueChange={(v) => setExpiry(v as Expiry)}>
                            <SelectTrigger className="w-full">
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="never">{t('api.create.expiryNever')}</SelectItem>
                                <SelectItem value="30">{t('api.create.expiryDays', { count: 30 })}</SelectItem>
                                <SelectItem value="90">{t('api.create.expiryDays', { count: 90 })}</SelectItem>
                                <SelectItem value="365">{t('api.create.expiryDays', { count: 365 })}</SelectItem>
                            </SelectContent>
                        </Select>
                    </div>
                </div>
            </ResponsiveDialog>

            <ResponsiveDialog
                open={createdToken !== null}
                onOpenChange={(open) => !open && setCreatedToken(null)}
                title={t('api.created.title')}
                description={t('api.created.description')}
                className="sm:max-w-lg"
                footer={<Button onClick={() => setCreatedToken(null)}>{t('api.created.done')}</Button>}
            >
                <div className="flex items-center gap-2">
                    <code className="min-w-0 flex-1 break-all rounded-md border bg-muted px-3 py-2 text-xs">
                        {createdToken}
                    </code>
                    <Button variant="outline" size="icon" onClick={handleCopy} aria-label={t('api.created.copy')}>
                        {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
                    </Button>
                </div>
            </ResponsiveDialog>
        </Page>
    )
}
