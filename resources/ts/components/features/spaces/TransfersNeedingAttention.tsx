import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { AlertTriangle, ArrowLeftRight } from 'lucide-react'
import { SettingsSection } from '@/components/shared'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useAccounts, useDeleteTransfer, useResolveTransfer, useSpaceTransfers, useUpdateTransfer } from '@/hooks'
import { transferReviewLabel } from '@/lib/labels'
import { Space, SpaceTransfer } from '@/types/spaces'

/** Transfers that need someone: pending after a restore, missing an account, unverifiable or frozen. */
export function TransfersNeedingAttention({ space }: { space: Space }) {
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

    const remoteAmount = (remote: NonNullable<SpaceTransfer['remote']>, outgoing: boolean) => {
        const side = outgoing ? remote.from : remote.to
        return `${side.amount} ${side.currency}`
    }

    const setAccount = (tr: SpaceTransfer) => {
        const id = Number(choice[tr.uuid])
        update.mutate({ uuid: tr.uuid, data: tr.outgoing ? { from_account_id: id } : { to_account_id: id } })
    }

    return (
        <SettingsSection
            icon={AlertTriangle}
            title={t('spaces.transfers.title')}
            description={t('spaces.transfers.description')}
        >
            <ul className="divide-y rounded-lg border">
                {items.map((tr) => (
                    <li key={tr.uuid} className="space-y-2 px-3 py-3 text-sm">
                        <div className="flex flex-wrap items-center justify-between gap-2">
                            <div className="flex min-w-0 items-center gap-2">
                                <ArrowLeftRight className="size-4 shrink-0 text-muted-foreground" />
                                <span className="truncate">
                                    {tr.description || t('spaces.transfers.untitled')} ·{' '}
                                    {tr.otherSpace?.name ?? t('spaces.transfers.unknownSpace')} · {tr.date}
                                </span>
                            </div>
                            <span className="font-medium tabular-nums">{amount(tr)}</span>
                        </div>
                        <div className="flex flex-wrap gap-1.5">
                            {tr.review && <Badge variant="outline">{transferReviewLabel(t, tr.review)}</Badge>}
                            {tr.status === 'needs_attention' && (
                                <Badge variant="outline">{t('spaces.transfers.needsAccount')}</Badge>
                            )}
                            {!tr.verified && <Badge variant="destructive">{t('spaces.transfers.unverified')}</Badge>}
                            {tr.frozen && <Badge variant="secondary">{t('spaces.transfers.frozen')}</Badge>}
                        </div>
                        {tr.remote && (
                            <p className="text-xs text-muted-foreground">
                                {tr.remote.deleted
                                    ? t('spaces.transfers.remoteDeleted')
                                    : t('spaces.transfers.remoteVersion', {
                                          amount: remoteAmount(tr.remote, tr.outgoing),
                                          date: tr.remote.date,
                                      })}
                            </p>
                        )}
                        <div className="flex flex-wrap gap-2">
                            {tr.status === 'pending' && (
                                <>
                                    <Button
                                        size="sm"
                                        onClick={() => resolve.mutate({ uuid: tr.uuid, decision: 'accept' })}
                                    >
                                        {t('spaces.transfers.accept')}
                                    </Button>
                                    <Button
                                        size="sm"
                                        variant="outline"
                                        disabled={!tr.verified && tr.review !== 'created_remote'}
                                        onClick={() => resolve.mutate({ uuid: tr.uuid, decision: 'reject' })}
                                    >
                                        {t('spaces.transfers.reject')}
                                    </Button>
                                </>
                            )}
                            {!tr.verified && (
                                <Button
                                    size="sm"
                                    variant="outline"
                                    onClick={() => resolve.mutate({ uuid: tr.uuid, decision: 'trust' })}
                                >
                                    {t('spaces.transfers.trust')}
                                </Button>
                            )}
                            {tr.status === 'needs_attention' && (
                                <>
                                    <Select
                                        value={choice[tr.uuid] ?? ''}
                                        onValueChange={(v) => setChoice({ ...choice, [tr.uuid]: v })}
                                    >
                                        <SelectTrigger
                                            className="h-8 w-48"
                                            aria-label={t('spaces.transfers.chooseAccount')}
                                        >
                                            <SelectValue placeholder={t('spaces.transfers.chooseAccount')} />
                                        </SelectTrigger>
                                        <SelectContent>
                                            {accounts.map((a) => (
                                                <SelectItem key={a.id} value={String(a.id)}>
                                                    {a.name}
                                                </SelectItem>
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
