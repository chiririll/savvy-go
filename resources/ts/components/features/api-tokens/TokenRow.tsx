import { useTranslation } from 'react-i18next'
import { Pencil, Trash2 } from 'lucide-react'
import { testId } from '@/lib/test-id'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
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
import { intlLocale } from '@/lib/i18n'
import { apiResourceLabel } from '@/lib/labels'
import type { ApiResource, ApiTokenSummary } from '@/types'
import { grantedResources } from './token-grants'

/** What a token may do, in words. */
function TokenGrants({ token }: { token: ApiTokenSummary }) {
    const { t } = useTranslation('settings')
    const { read, write, remove } = grantedResources(token)
    const list = (items: ApiResource[]) => items.map((r) => apiResourceLabel(t, r)).join(', ')
    if (read.length === 0 && write.length === 0 && remove.length === 0) {
        return <p className="text-xs text-muted-foreground">{t('api.tokens.noAccess')}</p>
    }
    return (
        <div className="space-y-0.5 text-xs text-muted-foreground">
            {read.length > 0 && <p>{t('api.tokens.canRead', { list: list(read) })}</p>}
            {write.length > 0 && <p>{t('api.tokens.canWrite', { list: list(write) })}</p>}
            {remove.length > 0 && <p>{t('api.tokens.canDelete', { list: list(remove) })}</p>}
        </div>
    )
}

interface TokenRowProps {
    token: ApiTokenSummary
    revoking: boolean
    onEdit: () => void
    onRevoke: () => void
}

export function TokenRow({ token, revoking, onEdit, onRevoke }: TokenRowProps) {
    const { t } = useTranslation('settings')
    const { t: tCommon } = useTranslation('common')
    const formatDate = (value: string) => new Date(value).toLocaleDateString(intlLocale())

    return (
        <div className="flex items-center justify-between gap-4 px-4 py-3.5">
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
            <div className="flex shrink-0 items-center">
                <Button
                    variant="ghost"
                    size="icon"
                    onClick={onEdit}
                    aria-label={t('api.tokens.editAria')}
                    {...testId(`edit-token-${token.id}`)}
                >
                    <Pencil className="size-4" />
                </Button>
                <AlertDialog>
                    <AlertDialogTrigger asChild>
                        <Button variant="ghost" size="icon" disabled={revoking} aria-label={t('api.tokens.revokeAria')}>
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
                            <AlertDialogAction onClick={onRevoke}>{t('api.tokens.revoke')}</AlertDialogAction>
                        </AlertDialogFooter>
                    </AlertDialogContent>
                </AlertDialog>
            </div>
        </div>
    )
}
