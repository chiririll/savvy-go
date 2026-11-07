import { Lock } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { parseDateKey } from '@/lib/dates'
import { intlLocale } from '@/lib/i18n'
import { formatCurrency } from '@/lib/utils'
import type { Transaction } from '@/types'
import { restrictionText } from './restriction-text'

/** Why the restricted form changes so little: the reason of each restriction, or that the form cannot show it all. */
export function LockedReasons({ transaction }: { transaction: Transaction }) {
    const { t } = useTranslation('forms')
    const reasons = transaction.restrictions.map((r) => r.reason)

    return (
        <Alert role="note">
            <Lock />
            <AlertTitle>{t('transactions.lockedTitle')}</AlertTitle>
            <AlertDescription>
                {reasons.length > 0 ? (
                    reasons.map((reason) => <p key={reason}>{restrictionText(t, reason)}</p>)
                ) : (
                    <p>{t('transactions.detailsOnly')}</p>
                )}
            </AlertDescription>
        </Alert>
    )
}

/** What the restricted form cannot change, as text: the date and each payment. */
export function LockedPayments({ transaction }: { transaction: Transaction }) {
    const { t } = useTranslation('forms')

    return (
        <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
            <dt className="text-muted-foreground">{t('transactions.lockedDate')}</dt>
            <dd className="text-right">
                {transaction.date
                    ? parseDateKey(transaction.date).toLocaleDateString(intlLocale())
                    : t('transactions.lockedNoDate')}
            </dd>
            {transaction.payments.map((p, index) => (
                <div key={`${p.accountId}-${index}`} className="col-span-2 flex min-w-0 justify-between gap-4">
                    <dt className="truncate text-muted-foreground">{p.account?.name ?? p.accountId}</dt>
                    <dd className="shrink-0 tabular-nums">{formatCurrency(p.amount, p.account?.currency)}</dd>
                </div>
            ))}
        </dl>
    )
}
