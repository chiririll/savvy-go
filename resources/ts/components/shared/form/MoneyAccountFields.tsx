import type { ReactNode } from 'react'
import type { Control, FieldPath, FieldValues } from 'react-hook-form'
import { useFormContext } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { ArrowUpDown } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { AccountSelect } from '../pickers/AccountSelect'
import { swapUpdates } from './transfer-swap'

interface MoneyAccountFieldsProps<T extends FieldValues> {
    control: Control<T>
    isTransfer: boolean
    accountId?: number | null
    fromCurrencySymbol?: string
    toCurrencySymbol?: string
    amountDisabled?: boolean
    /** Replaces the label of the amount, when it means something other than the whole amount. */
    amountLabel?: string
    /**
     * Sets a column after the account for a button of the row (add another
     * row), as wide as this; rows below it use the same width to line up.
     */
    actionWidth?: string
    rowAction?: ReactNode
    toAmountReadOnly?: boolean
    toAmountPlaceholder?: string
    onSwapAccounts?: () => void
    /**
     * Set when the destination account belongs to another space (null until
     * one is chosen). Such a transfer has one direction, so no swap button.
     */
    toSpaceId?: number | null
}

export function MoneyAccountFields<T extends FieldValues>({
    control,
    isTransfer,
    accountId,
    fromCurrencySymbol,
    toCurrencySymbol,
    amountDisabled,
    amountLabel,
    actionWidth,
    rowAction,
    toAmountReadOnly,
    toAmountPlaceholder,
    onSwapAccounts,
    toSpaceId,
}: MoneyAccountFieldsProps<T>) {
    const { t } = useTranslation(['common', 'forms'])
    const form = useFormContext<T>()
    const swappable = isTransfer && toSpaceId === undefined

    const swapAccounts = () => {
        const updates = swapUpdates(form.getValues() as Record<string, unknown>)
        if (!updates) {
            return
        }

        onSwapAccounts?.()

        const options = { shouldDirty: true, shouldValidate: false }
        for (const [name, value] of Object.entries(updates)) {
            form.setValue(name as FieldPath<T>, value as T[FieldPath<T>], options)
        }
    }

    return (
        <div className={swappable ? 'relative' : undefined}>
            <div className={cn(isTransfer && 'space-y-2', swappable && 'pr-12')}>
                <div
                    className={cn('grid gap-4', !actionWidth && 'grid-cols-2')}
                    style={actionWidth ? { gridTemplateColumns: `1fr 1fr ${actionWidth}` } : undefined}
                >
                    <FormField
                        control={control}
                        name={'amount' as FieldPath<T>}
                        render={({ field }) => (
                            <FormItem className="min-w-0">
                                <FormLabel>
                                    {amountLabel ?? (isTransfer ? t('forms:transactions.sendAmount') : t('fields.amount'))}
                                    {fromCurrencySymbol && (
                                        <span className="text-muted-foreground ml-1">({fromCurrencySymbol})</span>
                                    )}
                                </FormLabel>
                                <FormControl>
                                    <Input
                                        type="number"
                                        step="0.01"
                                        min={0}
                                        placeholder="0.00"
                                        {...field}
                                        disabled={amountDisabled}
                                    />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        )}
                    />

                    <FormField
                        control={control}
                        name={'account_id' as FieldPath<T>}
                        render={({ field }) => (
                            <FormItem className="min-w-0">
                                <FormLabel>{isTransfer ? t('forms:fromAccount') : t('fields.account')}</FormLabel>
                                <AccountSelect value={field.value} onChange={field.onChange} />
                                <FormMessage />
                            </FormItem>
                        )}
                    />
                    {actionWidth && <div className="mt-[1.375rem] flex justify-end gap-1">{rowAction}</div>}
                </div>

                {isTransfer && (
                    <div className="grid grid-cols-2 gap-4">
                        <FormField
                            control={control}
                            name={'to_amount' as FieldPath<T>}
                            render={({ field }) => (
                                <FormItem className="min-w-0">
                                    <FormLabel>
                                        {t('forms:transactions.receiveAmount')}
                                        {toCurrencySymbol && (
                                            <span className="text-muted-foreground ml-1">({toCurrencySymbol})</span>
                                        )}
                                    </FormLabel>
                                    <FormControl>
                                        <Input
                                            type="number"
                                            step="0.01"
                                            min={0}
                                            placeholder={toAmountPlaceholder ?? '0.00'}
                                            {...field}
                                            value={field.value ?? ''}
                                            onChange={(event) =>
                                                field.onChange(event.target.value ? Number(event.target.value) : null)
                                            }
                                            readOnly={toAmountReadOnly}
                                            disabled={toAmountReadOnly}
                                        />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={control}
                            name={'to_account_id' as FieldPath<T>}
                            render={({ field }) => (
                                <FormItem className="min-w-0">
                                    <FormLabel>{t('forms:transactions.toAccount')}</FormLabel>
                                    <AccountSelect
                                        value={field.value}
                                        onChange={field.onChange}
                                        excludeId={toSpaceId === undefined && accountId ? Number(accountId) : undefined}
                                        spaceId={toSpaceId}
                                    />
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                    </div>
                )}
            </div>

            {swappable && (
                <Button
                    type="button"
                    variant="outline"
                    size="icon-sm"
                    className="absolute right-0 top-1/2 -translate-y-1/2 rounded-full"
                    onClick={swapAccounts}
                    aria-label={t('forms:transactions.swapAccounts')}
                    title={t('forms:transactions.swapAccounts')}
                >
                    <ArrowUpDown className="size-4" />
                </Button>
            )}
        </div>
    )
}
