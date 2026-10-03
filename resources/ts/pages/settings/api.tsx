import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Check, Copy, ExternalLink, KeyRound, Plus, Trash2 } from 'lucide-react'
import { Page, PageHeader, FormWrapper } from '@/components/shared'
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
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog'
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
import { useApiTokens, useCreateApiToken, useRevokeApiToken } from '@/hooks'
import { useUser } from '@/stores/auth'
import { intlLocale } from '@/lib/i18n'
import type { ApiTokenScope } from '@/types'

type Expiry = 'never' | '30' | '90' | '365'

function expiryToIso(expiry: Expiry): string | null {
    if (expiry === 'never') return null
    const date = new Date()
    date.setDate(date.getDate() + Number(expiry))
    return date.toISOString()
}

export default function ApiSettingsPage() {
    const { t } = useTranslation('settings')
    const { t: tCommon } = useTranslation('common')
    const user = useUser()
    // Read-only users may still connect apps, but only with read access.
    const canWrite = user?.role !== 'read-only'

    const { data: tokens, isLoading } = useApiTokens()
    const createToken = useCreateApiToken()
    const revokeToken = useRevokeApiToken()

    const [showCreate, setShowCreate] = useState(false)
    const [name, setName] = useState('')
    const [scope, setScope] = useState<ApiTokenScope>('read')
    const [expiry, setExpiry] = useState<Expiry>('never')
    const [createdToken, setCreatedToken] = useState<string | null>(null)
    const [copied, setCopied] = useState(false)

    const formatDate = (value: string) => new Date(value).toLocaleDateString(intlLocale())

    const resetForm = () => {
        setName('')
        setScope('read')
        setExpiry('never')
    }

    const handleCreate = async () => {
        try {
            const created = await createToken.mutateAsync({
                name: name.trim(),
                scope: canWrite ? scope : 'read',
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
                            <Button size="sm" onClick={() => setShowCreate(true)}>
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
                                                <Badge variant={token.scope === 'read' ? 'secondary' : 'default'}>
                                                    {t(`api.scope.${token.scope}`)}
                                                </Badge>
                                                <code className="text-xs text-muted-foreground">{token.prefix}…</code>
                                            </div>
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

            <Dialog
                open={showCreate}
                onOpenChange={(open) => {
                    setShowCreate(open)
                    if (!open) resetForm()
                }}
            >
                <DialogContent className="sm:max-w-md">
                    <DialogHeader>
                        <DialogTitle>{t('api.create.title')}</DialogTitle>
                        <DialogDescription>{t('api.create.description')}</DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4 py-2">
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
                            <Label>{t('api.create.access')}</Label>
                            <Select value={scope} onValueChange={(v) => setScope(v as ApiTokenScope)}>
                                <SelectTrigger>
                                    <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="read">{t('api.scope.read')}</SelectItem>
                                    {canWrite && <SelectItem value="read-write">{t('api.scope.read-write')}</SelectItem>}
                                </SelectContent>
                            </Select>
                            <p className="text-xs text-muted-foreground">{t(`api.create.accessHint.${scope}`)}</p>
                        </div>
                        <div className="space-y-2">
                            <Label>{t('api.create.expiry')}</Label>
                            <Select value={expiry} onValueChange={(v) => setExpiry(v as Expiry)}>
                                <SelectTrigger>
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
                    <DialogFooter className="gap-2 sm:gap-0">
                        <Button variant="outline" onClick={() => setShowCreate(false)}>
                            {tCommon('actions.cancel')}
                        </Button>
                        <Button onClick={handleCreate} disabled={!name.trim() || createToken.isPending}>
                            {tCommon('actions.create')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            <Dialog open={createdToken !== null} onOpenChange={(open) => !open && setCreatedToken(null)}>
                <DialogContent className="sm:max-w-lg">
                    <DialogHeader>
                        <DialogTitle>{t('api.created.title')}</DialogTitle>
                        <DialogDescription>{t('api.created.description')}</DialogDescription>
                    </DialogHeader>
                    <div className="flex items-center gap-2 py-2">
                        <code className="min-w-0 flex-1 break-all rounded-md border bg-muted px-3 py-2 text-xs">
                            {createdToken}
                        </code>
                        <Button variant="outline" size="icon" onClick={handleCopy} aria-label={t('api.created.copy')}>
                            {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
                        </Button>
                    </div>
                    <DialogFooter>
                        <Button onClick={() => setCreatedToken(null)}>{t('api.created.done')}</Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </Page>
    )
}
