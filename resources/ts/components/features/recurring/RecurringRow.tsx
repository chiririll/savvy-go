import { useTranslation } from 'react-i18next'
import { FeedRow, RowActions, type FeedGroupKey } from '@/components/shared'
import { transactionIconProps } from '@/components/features/transactions/list/transaction-icon'
import { useRelativeDay } from '@/hooks'
import { pendingDateClassName } from '@/lib/dates'
import i18n from '@/lib/i18n'
import {
    displayTransactionDescription,
    transactionAmountAppearance,
    transactionSubtitle,
} from '@/lib/transaction-description'
import { cn, formatCurrency } from '@/lib/utils'
import { RecurringFrequency, RecurringTransaction, Transaction } from '@/types'
import { frequencyLabel } from '@/lib/labels'

/**
 * The rule still has the flat shape (F4f moves it to a template of payments and lines): this shows it
 * as the payments and lines the shared row texts read.
 */
function asPaymentsAndLines(recurring: RecurringTransaction): Pick<Transaction, 'payments' | 'lines'> {
    const sign = recurring.type === 'income' ? 1 : -1
    const payments = [{ accountId: recurring.account.id, account: recurring.account, amount: sign * recurring.amount }]
    if (recurring.toAccount) {
        payments.push({
            accountId: recurring.toAccount.id,
            account: recurring.toAccount,
            amount: recurring.toAmount ?? recurring.amount,
        })
    }
    const lines = recurring.category
        ? [{
              seq: 0,
              categoryId: recurring.category.id,
              category: recurring.category,
              currencyId: recurring.account.currency?.id ?? 0,
              amount: recurring.amount,
              note: null,
          }]
        : []
    return { payments, lines }
}

const FREQUENCY_ORDER: RecurringFrequency[] = ['daily', 'weekly', 'monthly', 'yearly']

/** Sections of the recurring list: one per schedule, shortest first. */
export function recurringGroup(recurring: RecurringTransaction): FeedGroupKey {
    const frequency = frequencyLabel(i18n.t, recurring.frequency)
    return {
        key: `${recurring.frequency}:${recurring.interval}`,
        title:
            recurring.interval === 1
                ? frequency
                : i18n.t('pages:recurring.everyInterval', {
                      count: recurring.interval,
                      frequency: frequency.toLowerCase(),
                  }),
    }
}

/** Schedule order, then active before paused, then the next run. */
export function compareRecurring(a: RecurringTransaction, b: RecurringTransaction): number {
    return (
        FREQUENCY_ORDER.indexOf(a.frequency) - FREQUENCY_ORDER.indexOf(b.frequency) ||
        a.interval - b.interval ||
        Number(b.isActive) - Number(a.isActive) ||
        a.nextRunDate.localeCompare(b.nextRunDate)
    )
}

interface RecurringRowProps {
    recurring: RecurringTransaction
    onEdit?: (recurring: RecurringTransaction) => void
    onDelete?: (id: number) => void
    isReadOnly?: boolean
}

export function RecurringRow({ recurring, onEdit, onDelete, isReadOnly }: RecurringRowProps) {
    const { t } = useTranslation('pages')
    const relativeDay = useRelativeDay()
    const canEdit = !!onEdit && !isReadOnly
    const canDelete = !!onDelete && !isReadOnly
    const { sign, className } = transactionAmountAppearance(recurring.type)
    const amount = formatCurrency(recurring.amount, recurring.account.currency)
    const shape = { ...asPaymentsAndLines(recurring), type: recurring.type, description: recurring.description }

    return (
        <FeedRow
            {...transactionIconProps(recurring.type, recurring.category)}
            title={displayTransactionDescription(shape)}
            subtitle={transactionSubtitle(shape)}
            amount={`${recurring.isEstimated ? '≈ ' : ''}${sign}${amount}`}
            amountClassName={className}
            extraAmount={
                <span
                    className={cn(
                        'font-sans',
                        recurring.isActive && pendingDateClassName(recurring.nextRunDate.slice(0, 10)),
                    )}
                >
                    {relativeDay(recurring.nextRunDate)}
                </span>
            }
            muted={!recurring.isActive}
            onOpen={canEdit ? () => onEdit(recurring) : undefined}
            hasActions={canEdit || canDelete}
            actions={({ menuOpen, setMenuOpen, isMobile }) => (
                <RowActions
                    open={menuOpen}
                    onOpenChange={setMenuOpen}
                    showTrigger={!isMobile}
                    onEdit={canEdit ? () => onEdit(recurring) : undefined}
                    onDelete={canDelete ? () => onDelete(recurring.id) : undefined}
                    deleteTitle={t('recurring.deleteTitle')}
                    deleteDescription={t('recurring.deleteDescription')}
                />
            )}
        />
    )
}
