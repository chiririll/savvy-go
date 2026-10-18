import { useCallback, useEffect, useRef, useState } from 'react'
import { useFieldArray, useWatch, type UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { FormField, FormMessage } from '@/components/ui/form'
import { formatCurrency, cn } from '@/lib/utils'
import { currencyDecimals, itemContribution, sumTransactionItems } from '@/lib/transaction-items'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Currency } from '@/types'

import { type ItemField } from './item-keys'
import { TransactionItemRow } from './TransactionItemRow'

interface TransactionFormItemsProps {
    form: UseFormReturn<TransactionFormValues>
    currency?: Currency | null
}

export function TransactionFormItems({ form, currency }: TransactionFormItemsProps) {
    const { t } = useTranslation(['common', 'forms'])
    const { fields, append, remove } = useFieldArray({
        control: form.control,
        name: 'items',
    })
    const items = useWatch({ control: form.control, name: 'items' })
    const decimals = currencyDecimals(currency)
    const [expandedIndex, setExpandedIndex] = useState<number | null>(null)
    const pendingNameFocus = useRef(false)
    const pendingExpandNew = useRef(false)

    const addItem = useCallback(() => {
        pendingNameFocus.current = true
        pendingExpandNew.current = true
        append({ name: '', quantity: 1, price_per_unit: 0 })
    }, [append])

    useEffect(() => {
        if (pendingExpandNew.current) {
            pendingExpandNew.current = false
            setExpandedIndex(fields.length > 0 ? fields.length - 1 : null)
        }

        if (!pendingNameFocus.current) {
            return
        }

        pendingNameFocus.current = false
        const inputs = document.querySelectorAll<HTMLInputElement>('[data-item-name]')
        inputs[inputs.length - 1]?.focus()
    }, [fields.length])

    const focusItemField = useCallback((index: number, field: ItemField) => {
        document.querySelector<HTMLInputElement>(`[data-item-${field}][data-index="${index}"]`)?.focus()
    }, [])

    const expandRow = useCallback((index: number) => {
        setExpandedIndex(index)
    }, [])

    const toggleRow = useCallback((index: number) => {
        setExpandedIndex((current) => (current === index ? null : index))
    }, [])

    return (
        <div className="min-w-0 space-y-3">
            <div className="flex min-w-0 items-center justify-between gap-2">
                <div className="text-sm font-medium leading-none">
                    {t('forms:transactions.itemsCount', { count: fields.length })}
                </div>
                <Button type="button" variant="outline" size="sm" onClick={addItem}>
                    <Plus className="size-4 mr-1" />
                    {t('forms:transactions.addItem')}
                </Button>
            </div>

            {fields.length > 0 && (
                <div className="min-w-0 space-y-2">
                    {fields.map((field, index) => (
                        <TransactionItemRow
                            key={field.id}
                            index={index}
                            form={form}
                            currency={currency}
                            decimals={decimals}
                            quantity={Number(items?.[index]?.quantity) || 0}
                            lineTotal={itemContribution(
                                Number(items?.[index]?.quantity) || 0,
                                Number(items?.[index]?.price_per_unit) || 0,
                                decimals,
                            )}
                            expanded={expandedIndex === index}
                            onExpand={() => expandRow(index)}
                            onToggle={() => toggleRow(index)}
                            canRemove={fields.length > 0}
                            onRemove={() => {
                                remove(index)
                                setExpandedIndex((current) => {
                                    if (current === null) {
                                        return null
                                    }
                                    if (current === index) {
                                        return null
                                    }
                                    return current > index ? current - 1 : current
                                })
                                requestAnimationFrame(() => {
                                    focusItemField(Math.max(0, index - 1), 'name')
                                })
                            }}
                            onAddItem={addItem}
                            onFocusField={focusItemField}
                            rowCount={fields.length}
                        />
                    ))}

                    <div
                        className={cn(
                            'flex min-w-0 items-center justify-end gap-2 rounded-lg border bg-muted/30 px-2',
                            'py-2 text-sm font-medium',
                        )}
                    >
                        <span>{t('forms:transactions.total')}:</span>
                        <span className="font-mono font-semibold">
                            {formatCurrency(sumTransactionItems(items, decimals), currency, { showSymbol: false })}
                        </span>
                    </div>
                </div>
            )}

            {fields.length === 0 && (
                <p className="text-sm text-muted-foreground text-center py-4 border rounded-lg border-dashed">
                    {t('forms:transactions.noItems')}
                </p>
            )}

            <FormField control={form.control} name="items" render={() => <FormMessage />} />
        </div>
    )
}
