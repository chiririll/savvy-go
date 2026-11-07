import { useEffect } from 'react'
import { type Control, type UseFieldArrayReturn, useFormContext, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { TriangleAlert, X } from 'lucide-react'
import { CategorySelect } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { formatCurrency } from '@/lib/utils'
import { linesNeedNotes, type LineCoverage } from '@/lib/transaction-lines'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Currency } from '@/types'
import { AmountInput } from './AmountInput'
import { LineNoteField } from './LineNoteField'

interface TransactionLinesProps {
    control: Control<TransactionFormValues>
    fields: UseFieldArrayReturn<TransactionFormValues, 'lines'>['fields']
    remove: UseFieldArrayReturn<TransactionFormValues, 'lines'>['remove']
    type: 'income' | 'expense'
    /** The currencies of the payment and change accounts: what a line can be in. */
    currencies: Currency[]
    coverage: LineCoverage
    /** What the lines leave uncovered, as text, when it is not nothing. */
    uncovered?: string | null
}

/** The lines the user wrote: a category and an amount each, a note from two lines on. */
export function TransactionLines({ control, fields, remove, type, currencies, coverage, uncovered }: TransactionLinesProps) {
    const { t } = useTranslation(['common', 'forms'])
    const { setValue } = useFormContext<TransactionFormValues>()
    const lines = useWatch({ control, name: 'lines' })
    const primary = coverage.currency
    const pickCurrency = currencies.length > 1
    const notes = linesNeedNotes(fields.length)

    // A line in a currency that the accounts no longer have goes back to the first payment's currency.
    useEffect(() => {
        lines?.forEach((line, index) => {
            if (line?.currency_id && !currencies.some((c) => c.id === Number(line.currency_id))) {
                setValue(`lines.${index}.currency_id`, null)
            }
        })
    }, [lines, currencies, setValue])

    if (fields.length === 0) {
        return null
    }

    // One place for what is off with the lines: they cover too much, or too little.
    const warning =
        coverage.excess > 0 && coverage.currency
            ? t('forms:transactions.linesExceed', { amount: formatCurrency(coverage.excess, coverage.currency) })
            : uncovered

    return (
        <div className="space-y-3" role="group" aria-label={t('forms:transactions.lines')}>
            {fields.map((field, index) => {
                const currencyId = Number(lines?.[index]?.currency_id) || primary?.id
                const currency = currencies.find((c) => c.id === currencyId) ?? primary
                const decimals = currency?.decimals ?? 2
                return (
                    <div key={field.id} className="space-y-1.5">
                        <div className="flex min-w-0 items-start gap-x-2 sm:gap-x-4 [&>*]:min-w-0 [&>*]:flex-1 [&>*]:basis-0">
                            <FormField
                                control={control}
                                name={`lines.${index}.category_id`}
                                render={({ field: f }) => (
                                    <FormItem className="min-w-0">
                                        {index === 0 && <FormLabel>{t('common:fields.category')}</FormLabel>}
                                        <CategorySelect
                                            value={f.value}
                                            onChange={f.onChange}
                                            type={type}
                                            ariaLabel={t('common:fields.category')}
                                        />
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={control}
                                name={`lines.${index}.amount`}
                                render={({ field: f }) => (
                                    <FormItem className="min-w-0">
                                        {index === 0 && <FormLabel>{t('common:fields.amount')}</FormLabel>}
                                        <div className="flex min-w-0 items-center gap-1">
                                            <div className="min-w-0 flex-1">
                                                <FormControl>
                                                    <AmountInput
                                                        aria-label={t('common:fields.amount')}
                                                        currency={currency}
                                                        hideCode={pickCurrency}
                                                        {...f}
                                                    />
                                                </FormControl>
                                            </div>
                                            {pickCurrency && (
                                                <FormField
                                                    control={control}
                                                    name={`lines.${index}.currency_id`}
                                                    render={({ field: c }) => (
                                                        <Select
                                                            value={String(currencyId ?? '')}
                                                            onValueChange={(v) => c.onChange(Number(v))}
                                                        >
                                                            <FormControl>
                                                                <SelectTrigger
                                                                    className="w-[4.75rem] shrink-0 px-2"
                                                                    aria-label={t('forms:transactions.lineCurrency')}
                                                                >
                                                                    <SelectValue />
                                                                </SelectTrigger>
                                                            </FormControl>
                                                            <SelectContent>
                                                                {currencies.map((option) => (
                                                                    <SelectItem key={option.id} value={String(option.id)}>
                                                                        {option.code}
                                                                    </SelectItem>
                                                                ))}
                                                            </SelectContent>
                                                        </Select>
                                                    )}
                                                />
                                            )}
                                            <Button
                                                type="button"
                                                variant="ghost"
                                                size="icon"
                                                className="size-8 shrink-0"
                                                aria-label={t('forms:transactions.removeLine')}
                                                title={t('forms:transactions.removeLine')}
                                                onClick={() => remove(index)}
                                            >
                                                <X />
                                            </Button>
                                        </div>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                        </div>
                        {notes && (
                            <FormField
                                control={control}
                                name={`lines.${index}.note`}
                                render={({ field: f }) => (
                                    <FormItem>
                                        <LineNoteField
                                            value={f.value ?? ''}
                                            onChange={f.onChange}
                                            onBlur={f.onBlur}
                                            label={t('forms:transactions.lineNote')}
                                        />
                                    </FormItem>
                                )}
                            />
                        )}
                    </div>
                )
            })}
            {warning && (
                <p role="status" className="flex items-center gap-1.5 text-xs text-warning">
                    <TriangleAlert className="size-3.5 shrink-0" />
                    {warning}
                </p>
            )}
        </div>
    )
}
