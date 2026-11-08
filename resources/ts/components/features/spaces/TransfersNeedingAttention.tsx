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
import { sideAmount, sidesForSpace, transferNeedsDecision } from './transfer-review'

/** Transfers that need someone: pending after a restore, missing an account, unverifiable or frozen. */
export function TransfersNeedingAttention({ space }: { space: Space }) {
    const { t } = useTranslation('settings')
    const { data: transfers = [] } = useSpaceTransfers(space.id)
    const { data: accounts = [] } = useAccounts()
    const resolve = useResolveTransfer(space.id)
    const update = useUpdateTransfer(space.id)
    const remove = useDeleteTransfer(space.id)
    const [choice, setChoice] = useState<Record<string, string>>({})
    const items = transfers.filter(transferNeedsDecision)

    if (!items.length) return null

    const amount = (tr: SpaceTransfer) => {
        const side = tr.outgoing ? tr.from : tr.to
        return sideAmount(side, tr.outgoing ? '−' : '+')
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
                            {tr.needsAttention && (
                                <Badge variant="outline">{t('spaces.transfers.needsAccount')}</Badge>
                            )}
                            {!tr.verified && <Badge variant="destructive">{t('spaces.transfers.unverified')}</Badge>}
                            {tr.frozen && <Badge variant="secondary">{t('spaces.transfers.frozen')}</Badge>}
                        </div>
                        {tr.status === 'in_review' && <TransferSides tr={tr} />}
                        <div className="flex flex-wrap gap-2">
                            {tr.status === 'in_review' && (
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
                            {tr.needsAttention && (
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
                            {tr.status !== 'in_review' && !tr.frozen && tr.verified && (
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

/** Both sides of a transfer in review: ours and the other space's version, each in its own currencies. */
function TransferSides({ tr }: { tr: SpaceTransfer }) {
    const { t } = useTranslation('settings')
    const [here, there] = sidesForSpace(tr, tr.outgoing)

    return (
        <table className="text-xs tabular-nums">
            <thead>
                <tr className="text-muted-foreground text-left font-normal">
                    <td />
                    <th scope="col" className="pr-4 font-normal">
                        {t('spaces.transfers.thisSpace')}
                    </th>
                    <th scope="col" className="font-normal">
                        {t('spaces.transfers.otherSpace')}
                    </th>
                </tr>
            </thead>
            <tbody>
                <tr>
                    <th scope="row" className="text-muted-foreground pr-4 text-left font-normal">
                        {t('spaces.transfers.ours')}
                    </th>
                    <td className="pr-4">{here}</td>
                    <td>{there}</td>
                </tr>
                {tr.remote && (
                    <tr>
                        <th scope="row" className="text-muted-foreground pr-4 text-left font-normal">
                            {t('spaces.transfers.remote')}
                        </th>
                        {tr.remote.deleted ? (
                            <td colSpan={2}>{t('spaces.transfers.remoteDeleted')}</td>
                        ) : (
                            sidesForSpace(tr.remote, tr.outgoing).map((side, i) => (
                                <td key={i} className={i === 0 ? 'pr-4' : undefined}>
                                    {side}
                                </td>
                            ))
                        )}
                    </tr>
                )}
            </tbody>
        </table>
    )
}
