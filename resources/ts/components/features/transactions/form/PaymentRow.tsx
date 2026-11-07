import type { ReactNode } from 'react'
import type { Control } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { AccountSelect } from '@/components/shared'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Currency } from '@/types'

interface PaymentRowProps {
    control: Control<TransactionFormValues>
    /** The list of the form the row belongs to: the payments, or the change. */
    list: 'payments' | 'change'
    index: number
    /** The currency of the chosen account, shown at the end of the amount. */
    currency?: Currency | null
    /** Accounts the other rows use; an account is in a transaction once. */
    excludeIds?: readonly number[]
    /** Replaces the labels, when the row means something other than a payment: a transfer's source. */
    amountLabel?: string
    accountLabel?: string
    /** A button at the end of the row: remove it, or add another. */
    action?: ReactNode
    /** The balance of the account after saving, drawn under the row. */
    children?: ReactNode
}

/** One payment: an amount (positive, in the currency of the account) and the account. Fits one line at 360px. */
export function PaymentRow({
    control,
    list,
    index,
    currency,
    excludeIds,
    amountLabel,
    accountLabel,
    action,
    children,
}: PaymentRowProps) {
    const { t } = useTranslation('common')
    const decimals = currency?.decimals ?? 2

    return (
        <div className="space-y-1">
            <div className="grid min-w-0 grid-cols-[minmax(6.5rem,0.8fr)_minmax(0,1fr)] gap-2 sm:gap-4">
                <FormField
                    control={control}
                    name={`${list}.${index}.amount`}
                    render={({ field }) => (
                        <FormItem className="min-w-0">
                            <FormLabel>{amountLabel ?? t('fields.amount')}</FormLabel>
                            <div className="relative">
                                <FormControl>
                                    <Input
                                        type="number"
                                        inputMode="decimal"
                                        step={decimals > 0 ? 10 ** -decimals : 1}
                                        min={0}
                                        placeholder="0.00"
                                        className={currency ? 'pr-12' : undefined}
                                        {...field}
                                        value={field.value ?? ''}
                                    />
                                </FormControl>
                                {currency && (
                                    <span className="pointer-events-none absolute inset-y-0 right-3 flex items-center text-xs text-muted-foreground">
                                        {currency.code}
                                    </span>
                                )}
                            </div>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <FormField
                    control={control}
                    name={`${list}.${index}.account_id`}
                    render={({ field }) => (
                        <FormItem className="min-w-0">
                            <FormLabel>{accountLabel ?? t('fields.account')}</FormLabel>
                            <div className="flex items-center gap-1">
                                <div className="min-w-0 flex-1">
                                    <AccountSelect value={field.value} onChange={field.onChange} excludeIds={excludeIds} />
                                </div>
                                {action}
                            </div>
                            <FormMessage />
                        </FormItem>
                    )}
                />
            </div>
            {children}
        </div>
    )
}
