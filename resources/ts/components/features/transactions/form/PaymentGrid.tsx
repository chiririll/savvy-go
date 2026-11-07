import { Plus, X } from 'lucide-react'
import type { ReactNode } from 'react'
import type { Control } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { AccountSelect } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { FormControl, FormField, FormItem, FormMessage } from '@/components/ui/form'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Account } from '@/types'
import type { BalancePreview } from './balance-preview'
import { AmountInput } from './AmountInput'
import { FORM_ROW } from './form-grid'
import { BalanceShort, type BalanceMode } from './TransactionBalanceHint'

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

const BUTTON = 'size-8 shrink-0'

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
            <div className={`${FORM_ROW} items-center`}>
                <div className="flex min-w-0 items-baseline justify-between gap-2">
                    <span className="text-sm leading-none font-medium">{t('fields.amount')}</span>
                    {summary}
                </div>
                <div className="flex items-center justify-between gap-1">
                    <span className="text-sm leading-none font-medium">{t('fields.account')}</span>
                    {/* A disabled button takes no hover, so the reason sits on the wrapper. */}
                    <span className="-my-1.5" title={addDisabled ? addDisabledReason : addLabel}>
                        <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            className={BUTTON}
                            aria-label={addLabel}
                            disabled={addDisabled}
                            onClick={onAdd}
                        >
                            <Plus />
                        </Button>
                    </span>
                </div>
            </div>

            {keys.map((key, index) => {
                const account = accounts?.find((a) => a.id === Number(rows?.[index]?.account_id))
                const preview = balances.find((row) => row.account.id === account?.id)
                return (
                    <div key={key} className="space-y-0.5">
                        <div className={FORM_ROW}>
                            <FormField
                                control={control}
                                name={`${list}.${index}.amount`}
                                render={({ field }) => (
                                    <FormItem>
                                        <FormControl>
                                            <AmountInput
                                                aria-label={t('fields.amount')}
                                                currency={account?.currency}
                                                {...field}
                                            />
                                        </FormControl>

                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={control}
                                name={`${list}.${index}.account_id`}
                                render={({ field }) => (
                                    <FormItem>
                                        <div className="flex items-center gap-1">
                                            <div className="min-w-0 flex-1">
                                                <AccountSelect
                                                    value={field.value}
                                                    onChange={field.onChange}
                                                    excludeIds={usedAccountIds.filter((id) => id !== account?.id)}
                                                    ariaLabel={t('fields.account')}
                                                />
                                            </div>
                                            {onRemove && (
                                                <Button
                                                    type="button"
                                                    variant="ghost"
                                                    size="icon"
                                                    className={BUTTON}
                                                    aria-label={removeLabel}
                                                    title={removeLabel}
                                                    onClick={() => onRemove(index)}
                                                >
                                                    <X />
                                                </Button>
                                            )}
                                        </div>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                        </div>
                        {/* Always there, even with no balance, so a message does not move the form. */}
                        <div className={FORM_ROW}>
                            {/* What is wrong with the amount, in the line of the balance. */}
                            <FormField
                                control={control}
                                name={`${list}.${index}.amount`}
                                render={() => (
                                    <FormItem className="min-h-5">
                                        <FormMessage role="alert" className="truncate text-xs leading-5" />
                                    </FormItem>
                                )}
                            />
                            {balanceMode !== 'none' ? (
                                <BalanceShort account={account} preview={preview} mode={balanceMode} />
                            ) : (
                                <span className="min-h-5" />
                            )}
                        </div>
                    </div>
                )
            })}
        </div>
    )
}
