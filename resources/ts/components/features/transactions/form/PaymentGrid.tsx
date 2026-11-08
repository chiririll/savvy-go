import { Plus, X } from 'lucide-react'
import type { ReactNode } from 'react'
import type { Control } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Account } from '@/types'
import type { BalancePreview } from './balance-preview'
import { AmountAccountRow, ROW_BUTTON, RowLabels } from './AmountAccountRow'
import { BalanceShort, type BalanceMode } from './TransactionBalanceHint'
import { testId } from '@/lib/test-id'

interface PaymentGridProps {
    control: Control<TransactionFormValues>
    /** The list of the form the rows belong to: the payments, or the change. */
    list: 'payments' | 'change'
    /** Keys of the rows, from the field array. */
    keys: string[]
    /** The account each row has now. */
    rows?: Array<{ account_id?: number | string | null } | undefined>
    accounts?: Account[]
    usedAccountIds: number[]
    balances: BalancePreview[]
    balanceMode: BalanceMode
    onAdd: () => void
    addLabel: string
    addDisabled?: boolean
    addDisabledReason?: string
    /** Removes a row; left out when the only row has to stay. */
    onRemove?: (index: number) => void
    removeLabel: string
    /** Small text at the end of the Amount label: the totals. */
    summary?: ReactNode
}

/**
 * The accounts that paid (or got the change), in two columns: the amount, and the account with the (X) of its row
 * inside it. The labels are written once, the (+) that adds a row at the end of them. Under each account stands
 * what it holds, so nothing moves when an amount is typed.
 */
export function PaymentGrid({
    control,
    list,
    keys,
    rows,
    accounts,
    usedAccountIds,
    balances,
    balanceMode,
    onAdd,
    addLabel,
    addDisabled,
    addDisabledReason,
    onRemove,
    removeLabel,
    summary,
}: PaymentGridProps) {
    const { t } = useTranslation('common')

    return (
        <div className="space-y-1.5">
            <RowLabels
                amount={t('fields.amount')}
                amountEnd={summary}
                account={t('fields.account')}
                accountEnd={
                    // A disabled button takes no hover, so the reason sits on the wrapper.
                    <span className="-my-1.5" title={addDisabled ? addDisabledReason : addLabel}>
                        <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            className={ROW_BUTTON}
                            aria-label={addLabel}
                            disabled={addDisabled}
                            onClick={onAdd}
                            {...testId(`tx-add-${list}`)}
                        >
                            <Plus />
                        </Button>
                    </span>
                }
            />

            {keys.map((key, index) => {
                const account = accounts?.find((a) => a.id === Number(rows?.[index]?.account_id))
                const preview = balances.find((row) => row.account.id === account?.id)
                return (
                    <AmountAccountRow
                        key={key}
                        control={control}
                        amountName={`${list}.${index}.amount`}
                        accountName={`${list}.${index}.account_id`}
                        amountLabel={t('fields.amount')}
                        accountLabel={t('fields.account')}
                        account={account}
                        excludeIds={usedAccountIds.filter((id) => id !== account?.id)}
                        action={
                            onRemove && (
                                <Button
                                    type="button"
                                    variant="ghost"
                                    size="icon"
                                    className={ROW_BUTTON}
                                    aria-label={removeLabel}
                                    title={removeLabel}
                                    onClick={() => onRemove(index)}
                                >
                                    <X />
                                </Button>
                            )
                        }
                        balance={balanceMode !== 'none' ? <BalanceShort account={account} preview={preview} mode={balanceMode} /> : undefined}
                    />
                )
            })}
        </div>
    )
}
