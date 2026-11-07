import { Plus, X } from 'lucide-react'
import { useFormContext, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { CategorySelect } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Currency } from '@/types'

interface TransferFeeSectionProps {
    /** The currency of the source account: the fee is paid in it. */
    currency?: Currency | null
}

/**
 * A commission the source pays on top of the amount. It is saved as a line in an expense category, so reports
 * count it as spending. Same-currency transfers use it too: "pay the card" with the bank's commission.
 */
export function TransferFeeSection({ currency }: TransferFeeSectionProps) {
    const { t } = useTranslation(['common', 'forms'])
    const form = useFormContext<TransactionFormValues>()
    const fee = useWatch({ control: form.control, name: 'fee' })
    const decimals = currency?.decimals ?? 2

    if (!fee) {
        return (
            <section className="flex items-center justify-between gap-2" aria-label={t('forms:transactions.feeTitle')}>
                <div className="min-w-0">
                    <div className="text-sm font-medium">{t('forms:transactions.feeTitle')}</div>
                    <p className="text-xs text-muted-foreground">{t('forms:transactions.feeHelp')}</p>
                </div>
                <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="size-8 shrink-0"
                    aria-label={t('forms:transactions.addFee')}
                    title={t('forms:transactions.addFee')}
                    onClick={() => form.setValue('fee', { amount: '' as unknown as number, category_id: null })}
                >
                    <Plus />
                </Button>
            </section>
        )
    }

    return (
        <section className="space-y-2" aria-label={t('forms:transactions.feeTitle')}>
            <div className="text-sm font-medium">{t('forms:transactions.feeTitle')}</div>
            <div className="grid min-w-0 grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] items-start gap-2 sm:gap-4">
                <FormField
                    control={form.control}
                    name="fee.category_id"
                    render={({ field }) => (
                        <FormItem className="min-w-0">
                            <FormLabel>{t('common:fields.category')}</FormLabel>
                            <CategorySelect
                                value={field.value}
                                onChange={field.onChange}
                                type="expense"
                                ariaLabel={t('forms:transactions.feeCategory')}
                            />
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <FormField
                    control={form.control}
                    name="fee.amount"
                    render={({ field }) => (
                        <FormItem className="min-w-0">
                            <FormLabel>{t('common:fields.amount')}</FormLabel>
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
                <div className="pt-[1.375rem]">
                    <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        aria-label={t('forms:transactions.removeFee')}
                        title={t('forms:transactions.removeFee')}
                        onClick={() => form.setValue('fee', null)}
                    >
                        <X />
                    </Button>
                </div>
            </div>
        </section>
    )
}
