import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, Trash2 } from 'lucide-react'
import { Confirm } from '@/components/shared/dialogs/Confirm'
import { ResponsiveDialog } from '@/components/shared/dialogs/ResponsiveDialog'
import { Button } from '@/components/ui/button'
import { formatCurrency } from '@/lib/currency'
import { cn } from '@/lib/utils'
import { signedAmount } from './reconciliation-days'
import type { Account, ReconciliationDay, ReconciliationPoint } from '@/types'

interface ReconciliationDetailsDialogProps {
    day: ReconciliationDay | null
    accounts?: readonly Account[]
    /** Shown for a date as the heading, already localised. */
    dateLabel: string
    onOpenChange: (open: boolean) => void
    /** Left out for a user who may not change data: no Delete action. */
    onDelete?: (id: number) => void
}

export function ReconciliationDetailsDialog({
    day,
    accounts = [],
    dateLabel,
    onOpenChange,
    onDelete,
}: ReconciliationDetailsDialogProps) {
    const { t } = useTranslation('pages')
    const [removing, setRemoving] = useState<ReconciliationPoint | null>(null)
    const accountOf = (point: ReconciliationPoint) => accounts.find((account) => account.id === point.accountId)
    const nameOf = (point: ReconciliationPoint) => accountOf(point)?.name ?? `#${point.accountId}`

    return (
        <>
            <ResponsiveDialog
                open={day !== null}
                onOpenChange={onOpenChange}
                title={t('transactions.reconciled.title', { date: dateLabel })}
                description={t('transactions.reconciled.explain')}
            >
                <ul className="divide-y divide-border/60">
                    {day?.points.map((point) => {
                        const currency = accountOf(point)?.currency
                        const money = (value: number) => formatCurrency(value, currency)
                        const deleteButton = onDelete && (
                            <Button
                                variant="ghost"
                                size="icon"
                                className="size-7 text-muted-foreground"
                                aria-label={t('transactions.reconciled.delete')}
                                onClick={() => setRemoving(point)}
                            >
                                <Trash2 className="size-4" />
                            </Button>
                        )
                        if (point.difference === 0) {
                            // It agrees with the ledger: one line, nothing to look into.
                            return (
                                <li key={point.id} className="py-2 text-sm first:pt-0 last:pb-0">
                                    <div className="flex items-center gap-3">
                                        <span className="min-w-0 flex-1 truncate font-medium">{nameOf(point)}</span>
                                        <span className="tabular-nums">{money(point.actualBalance)}</span>
                                        <Check
                                            className="size-4 shrink-0 text-success"
                                            role="img"
                                            aria-label={t('transactions.reconciled.matches')}
                                        />
                                        {deleteButton ?? <span className="size-7 shrink-0" aria-hidden />}
                                    </div>
                                    {point.note && <p className="text-muted-foreground">{point.note}</p>}
                                </li>
                            )
                        }
                        return (
                            <li key={point.id} className="space-y-1 py-3 text-sm first:pt-0 last:pb-0">
                                <div className="flex items-center justify-between gap-2">
                                    <span className="font-medium">{nameOf(point)}</span>
                                    {deleteButton}
                                </div>
                                <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-0.5 tabular-nums">
                                    <dt className="text-muted-foreground">{t('transactions.reconciled.actual')}</dt>
                                    <dd className="text-right">{money(point.actualBalance)}</dd>
                                    <dt className="text-muted-foreground">{t('transactions.reconciled.expected')}</dt>
                                    <dd className="text-right">{money(point.expectedBalance)}</dd>
                                    <dt className="text-muted-foreground">{t('transactions.reconciled.difference')}</dt>
                                    <dd
                                        className={cn(
                                            'text-right font-medium',
                                            point.difference > 0 && 'text-success',
                                            point.difference < 0 && 'text-destructive',
                                        )}
                                    >
                                        {signedAmount(point.difference, money)}
                                    </dd>
                                </dl>
                                {point.note && <p className="text-muted-foreground">{point.note}</p>}
                            </li>
                        )
                    })}
                </ul>
            </ResponsiveDialog>
            <Confirm
                open={removing !== null}
                onOpenChange={(open) => !open && setRemoving(null)}
                title={t('transactions.reconciled.deleteTitle')}
                description={
                    removing &&
                    t('transactions.reconciled.deleteDescription', { name: nameOf(removing), date: dateLabel })
                }
                action={t('common:actions.delete')}
                destructive
                onConfirm={() => removing && onDelete?.(removing.id)}
            />
        </>
    )
}
