import type { ReactNode } from 'react'
import type { Control, FieldPath } from 'react-hook-form'
import { AccountSelect } from '@/components/shared'
import { FormControl, FormField, FormItem, FormMessage } from '@/components/ui/form'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Account } from '@/types'
import { AmountInput } from './AmountInput'
import { FORM_ROW } from './form-grid'

type Name = FieldPath<TransactionFormValues>

/** The size of the button at the end of a row; a row without one keeps the room so the accounts line up. */
export const ROW_BUTTON = 'size-8 shrink-0'

interface RowLabelsProps {
    amount: ReactNode
    account: ReactNode
    /** Small text at the end of the amount label: the totals. */
    amountEnd?: ReactNode
    /** At the end of the account label: the button that adds a row. */
    accountEnd?: ReactNode
}

/** The labels over a row, in the same two halves as the row. */
export function RowLabels({ amount, account, amountEnd, accountEnd }: RowLabelsProps) {
    return (
        <div className={`${FORM_ROW} items-center`}>
            <div className="flex min-w-0 items-baseline justify-between gap-2">
                <span className="text-sm leading-none font-medium">{amount}</span>
                {amountEnd}
            </div>
            <div className="flex items-center justify-between gap-1">
                <span className="text-sm leading-none font-medium">{account}</span>
                {accountEnd}
            </div>
        </div>
    )
}

interface AmountAccountRowProps {
    control: Control<TransactionFormValues>
    amountName: Name
    accountName: Name
    amountLabel: string
    accountLabel: string
    /** The account the row has now: its currency is shown in the amount. */
    account?: Account
    excludeIds?: readonly number[]
    excludeId?: number | null
    spaceId?: number | null
    /** Placed after the account: remove the row, or swap. */
    action?: ReactNode
    /** Shown in the amount instead of a field of the form; the amount cannot be changed. */
    fixedAmount?: string | number | null
    amountPlaceholder?: string
    /** The text typed in the amount, before it goes to the form. */
    parseAmount?: (value: string) => unknown
    /** Under the account: its balance. */
    balance?: ReactNode
}

/**
 * An amount and an account of the form, in two halves of the same width as every other row of it: the payments, the
 * change and the two sides of a transfer. Under it stands a line for what is wrong with the amount and for the
 * balance. It is there even when both are empty, so a message does not move the form.
 */
export function AmountAccountRow({
    control,
    amountName,
    accountName,
    amountLabel,
    accountLabel,
    account,
    excludeIds,
    excludeId,
    spaceId,
    action,
    fixedAmount,
    amountPlaceholder,
    parseAmount,
    balance,
}: AmountAccountRowProps) {
    const fixed = fixedAmount !== undefined

    return (
        <div className="space-y-0.5">
            <div className={FORM_ROW}>
                {fixed ? (
                    <FormItem>
                        <AmountInput aria-label={amountLabel} currency={account?.currency} value={fixedAmount} onChange={() => {}} disabled />
                    </FormItem>
                ) : (
                    <FormField
                        control={control}
                        name={amountName}
                        render={({ field }) => (
                            <FormItem>
                                <FormControl>
                                    <AmountInput
                                        aria-label={amountLabel}
                                        currency={account?.currency}
                                        placeholder={amountPlaceholder}
                                        name={field.name}
                                        ref={field.ref}
                                        onBlur={field.onBlur}
                                        value={field.value as string | number | null}
                                        onChange={(value) => field.onChange(parseAmount ? parseAmount(value) : value)}
                                    />
                                </FormControl>
                            </FormItem>
                        )}
                    />
                )}
                <FormField
                    control={control}
                    name={accountName}
                    render={({ field }) => (
                        <FormItem>
                            <div className="flex items-center gap-1">
                                <div className="min-w-0 flex-1">
                                    <AccountSelect
                                        value={field.value as number | null}
                                        onChange={field.onChange}
                                        excludeIds={excludeIds}
                                        excludeId={excludeId}
                                        spaceId={spaceId}
                                        ariaLabel={accountLabel}
                                        testName={accountName}
                                    />
                                </div>
                                {action}
                            </div>
                            <FormMessage />
                        </FormItem>
                    )}
                />
            </div>
            <div className={FORM_ROW}>
                {fixed ? (
                    <span className="min-h-5" />
                ) : (
                    <FormField
                        control={control}
                        name={amountName}
                        render={() => (
                            <FormItem className="min-h-5">
                                <FormMessage role="alert" className="truncate text-xs leading-5" />
                            </FormItem>
                        )}
                    />
                )}
                {balance ?? <span className="min-h-5" />}
            </div>
        </div>
    )
}
