import { useFieldArray, useWatch, type UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Plus, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { AccountSelect } from '@/components/shared'
import { useAccounts } from '@/hooks'
import { buildPaymentLegs } from '@/lib/payment-input'
import { rateDeviation } from '@/lib/split-payment'
import { formatCurrency } from '@/lib/utils'
import type { PaymentMode, TransactionFormValues } from '@/schemas/transactions'
import type { Account } from '@/types'
import { ACCOUNT_FILTER } from './use-transaction-model'

/** A rate this far from the stored one is probably a typo. */
const RATE_WARNING = 0.1

interface PaymentSectionProps {
    form: UseFormReturn<TransactionFormValues>
    /** A saved payment: it stays a split one. */
    paymentEdit?: boolean
}

const symbolOf = (account?: Account) => account?.currency?.symbol

function amountInput(
    field: { value?: number | null; onChange: (value: number | null) => void },
    placeholder = '0.00',
) {
    return (
        <Input
            type="number"
            step="0.01"
            min={0}
            placeholder={placeholder}
            value={field.value ?? ''}
            onChange={(event) => field.onChange(event.target.value ? Number(event.target.value) : null)}
        />
    )
}

/** How an expense was paid: one account, several, or with change given back in another currency. */
export function PaymentSection({ form, paymentEdit }: PaymentSectionProps) {
    const { t } = useTranslation(['common', 'forms'])
    const mode = useWatch({ control: form.control, name: 'payment_mode' }) ?? 'single'

    const setMode = (next: PaymentMode) => {
        form.setValue('payment_mode', next)
        form.setValue('items', [])
        form.setValue('is_estimated', false)
        form.setValue('extra_payments', next === 'several' ? [{ account_id: 0, amount: 0 }] : [])
        form.setValue('handed_over', null)
        form.setValue('change_account_id', null)
        form.setValue('change_amount', null)
        form.clearErrors(['extra_payments', 'handed_over', 'change_account_id', 'change_amount'])
    }

    const labels: Record<PaymentMode, string> = {
        single: t('forms:transactions.payment.mode.single'),
        several: t('forms:transactions.payment.mode.several'),
        change: t('forms:transactions.payment.mode.change'),
    }
    const modes: PaymentMode[] = paymentEdit ? ['several', 'change'] : ['single', 'several', 'change']

    return (
        <div className="min-w-0 space-y-3">
            <div className="grid grid-cols-2 items-center gap-4">
                <span className="text-sm font-medium">{t('forms:transactions.payment.label')}</span>
                <Select value={mode} onValueChange={(value) => setMode(value as PaymentMode)}>
                    <SelectTrigger aria-label={t('forms:transactions.payment.label')}>
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                        {modes.map((option) => (
                            <SelectItem key={option} value={option}>
                                {labels[option]}
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>
            </div>

            {mode === 'several' && <SeveralAccounts form={form} />}
            {mode === 'change' && <PaidWithChange form={form} />}
        </div>
    )
}

function SeveralAccounts({ form }: { form: UseFormReturn<TransactionFormValues> }) {
    const { t } = useTranslation(['common', 'forms'])
    const { data: accounts } = useAccounts(ACCOUNT_FILTER)
    const { fields, append, remove } = useFieldArray({ control: form.control, name: 'extra_payments' })
    const [accountId, amount, extra] = useWatch({
        control: form.control,
        name: ['account_id', 'amount', 'extra_payments'],
    })

    // One total per currency: amounts of different currencies are not added.
    const totals = new Map<string, { amount: number; account: Account }>()
    for (const part of [{ account_id: accountId, amount }, ...(extra ?? [])]) {
        const account = accounts?.find((a) => a.id === Number(part.account_id))
        if (account?.currency) {
            const key = account.currency.code
            totals.set(key, { account, amount: (totals.get(key)?.amount ?? 0) + (Number(part.amount) || 0) })
        }
    }

    return (
        <div className="min-w-0 space-y-3 rounded-lg border bg-muted/20 p-3">
            <p className="text-xs text-muted-foreground">{t('forms:transactions.payment.severalHelp')}</p>
            {fields.map((field, index) => (
                <div key={field.id} className="grid grid-cols-[1fr_1fr_auto] items-start gap-2">
                    <FormField
                        control={form.control}
                        name={`extra_payments.${index}.amount`}
                        render={({ field: amountField }) => (
                            <FormItem className="min-w-0">
                                <FormControl>{amountInput(amountField)}</FormControl>
                                <FormMessage />
                            </FormItem>
                        )}
                    />
                    <FormField
                        control={form.control}
                        name={`extra_payments.${index}.account_id`}
                        render={({ field: accountField }) => (
                            <FormItem className="min-w-0">
                                <AccountSelect value={accountField.value} onChange={accountField.onChange} />
                                <FormMessage />
                            </FormItem>
                        )}
                    />
                    <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        aria-label={t('forms:transactions.payment.removeAccount')}
                        title={t('forms:transactions.payment.removeAccount')}
                        onClick={() => remove(index)}
                    >
                        <X className="size-4" />
                    </Button>
                </div>
            ))}

            <FormField control={form.control} name="extra_payments" render={() => <FormMessage />} />

            <div className="flex min-w-0 flex-wrap items-center justify-between gap-2">
                <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => append({ account_id: 0, amount: 0 })}
                >
                    <Plus className="mr-1 size-4" />
                    {t('forms:transactions.payment.addAccount')}
                </Button>
                <span className="text-sm font-medium">
                    {t('forms:transactions.total')}:{' '}
                    <span className="font-mono">
                        {[...totals.values()]
                            .map((total) => formatCurrency(total.amount, total.account.currency))
                            .join(' + ')}
                    </span>
                </span>
            </div>
        </div>
    )
}

function PaidWithChange({ form }: { form: UseFormReturn<TransactionFormValues> }) {
    const { t } = useTranslation(['common', 'forms'])
    const { data: accounts } = useAccounts(ACCOUNT_FILTER)
    const [accountId, changeAccountId] = useWatch({ control: form.control, name: ['account_id', 'change_account_id'] })
    // Re-evaluated on every change of the values it reads.
    useWatch({ control: form.control, name: ['amount', 'handed_over', 'change_amount'] })
    const paidWith = accounts?.find((a) => a.id === Number(accountId))
    const changeTo = accounts?.find((a) => a.id === Number(changeAccountId))

    const built = accounts ? buildPaymentLegs({ ...form.getValues(), payment_mode: 'change' }, accounts) : null
    const exchange = built?.ok ? built.legs[1] : null
    const rate = built?.ok ? built.rate : undefined
    const stored =
        paidWith?.currency?.rate && changeTo?.currency?.rate ? paidWith.currency.rate / changeTo.currency.rate : null
    const deviation = rate !== undefined ? rateDeviation(rate, stored) : null

    return (
        <div className="min-w-0 space-y-3 rounded-lg border bg-muted/20 p-3">
            <p className="text-xs text-muted-foreground">{t('forms:transactions.payment.changeHelp')}</p>
            <div className="grid grid-cols-2 gap-4">
                <FormField
                    control={form.control}
                    name="handed_over"
                    render={({ field }) => (
                        <FormItem className="min-w-0">
                            <FormLabel>
                                {t('forms:transactions.payment.handedOver')}
                                {symbolOf(paidWith) && (
                                    <span className="ml-1 text-muted-foreground">({symbolOf(paidWith)})</span>
                                )}
                            </FormLabel>
                            <FormControl>{amountInput(field)}</FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <FormField
                    control={form.control}
                    name="change_amount"
                    render={({ field }) => (
                        <FormItem className="min-w-0">
                            <FormLabel>
                                {t('forms:transactions.payment.changeAmount')}
                                {symbolOf(changeTo) && (
                                    <span className="ml-1 text-muted-foreground">({symbolOf(changeTo)})</span>
                                )}
                            </FormLabel>
                            <FormControl>{amountInput(field)}</FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
            </div>
            <FormField
                control={form.control}
                name="change_account_id"
                render={({ field }) => (
                    <FormItem className="min-w-0">
                        <FormLabel>{t('forms:transactions.payment.changeAccount')}</FormLabel>
                        <AccountSelect value={field.value} onChange={field.onChange} />
                        <FormMessage />
                    </FormItem>
                )}
            />
            {exchange && rate !== undefined && paidWith && changeTo && (
                <p className="text-sm text-muted-foreground">
                    {t('forms:transactions.payment.exchanged', {
                        from: formatCurrency(exchange.amount, paidWith.currency),
                        to: formatCurrency(exchange.to_amount ?? 0, changeTo.currency),
                        rate,
                    })}
                    {deviation !== null && deviation > RATE_WARNING && (
                        <span className="ml-1 text-amber-600 dark:text-amber-400">
                            {t('forms:transactions.payment.rateOff', { percent: Math.round(deviation * 100) })}
                        </span>
                    )}
                </p>
            )}
        </div>
    )
}
