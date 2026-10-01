import { useCallback, useEffect, useLayoutEffect, useRef, useState, type KeyboardEvent } from 'react'
import { useFieldArray, useFormContext, useWatch, type UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { ChevronDown, Plus, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
} from '@/components/ui/form'
import { cn, formatCurrency } from '@/lib/utils'
import {
    ITEM_QTY_DECIMALS,
    currencyDecimals,
    itemContribution,
    priceInputStep,
    sumTransactionItems,
} from '@/lib/transaction-items'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Currency } from '@/types'

type ItemField = 'name' | 'quantity' | 'price_per_unit'

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
        document.querySelector<HTMLInputElement>(
            `[data-item-${field}][data-index="${index}"]`,
        )?.focus()
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

                    <div className="flex min-w-0 items-center justify-end gap-2 rounded-lg border bg-muted/30 px-2 py-2 text-sm font-medium">
                        <span>{t('forms:transactions.total')}:</span>
                        <span className="font-mono font-semibold">
                            {formatCurrency(
                                sumTransactionItems(items, decimals),
                                currency,
                                { showSymbol: false },
                            )}
                        </span>
                    </div>
                </div>
            )}

            {fields.length === 0 && (
                <p className="text-sm text-muted-foreground text-center py-4 border rounded-lg border-dashed">
                    {t('forms:transactions.noItems')}
                </p>
            )}

            <FormField
                control={form.control}
                name="items"
                render={() => <FormMessage />}
            />
        </div>
    )
}

function TransactionItemRow({
    index,
    form,
    currency,
    decimals,
    lineTotal,
    expanded,
    onExpand,
    onToggle,
    canRemove,
    onRemove,
    onAddItem,
    onFocusField,
    rowCount,
}: {
    index: number
    form: UseFormReturn<TransactionFormValues>
    currency?: Currency | null
    decimals: number
    lineTotal: number
    expanded: boolean
    onExpand: () => void
    onToggle: () => void
    canRemove: boolean
    onRemove: () => void
    onAddItem: () => void
    onFocusField: (index: number, field: ItemField) => void
    rowCount: number
}) {
    const { t } = useTranslation(['common', 'forms'])
    const { getFieldState, formState } = useFormContext<TransactionFormValues>()
    const nameRef = useRef<HTMLInputElement | null>(null)
    const quantityRef = useRef<HTMLInputElement | null>(null)
    const priceRef = useRef<HTMLInputElement | null>(null)
    const pendingPriceFocus = useRef(false)

    const quantityState = getFieldState(`items.${index}.quantity`, formState)
    const priceState = getFieldState(`items.${index}.price_per_unit`, formState)

    useEffect(() => {
        if (quantityState.error || priceState.error) {
            onExpand()
        }
    }, [onExpand, quantityState.error, priceState.error])

    useLayoutEffect(() => {
        if (!expanded || !pendingPriceFocus.current) {
            return
        }

        pendingPriceFocus.current = false
        priceRef.current?.focus()
    }, [expanded])

    const expandToPrice = useCallback(() => {
        if (expanded) {
            priceRef.current?.focus()
            return
        }

        pendingPriceFocus.current = true
        onExpand()
    }, [expanded, onExpand])

    const handleItemKeyDown = useCallback((
        event: KeyboardEvent<HTMLInputElement>,
        field: ItemField,
    ) => {
        if (event.key === 'Enter') {
            event.preventDefault()
            onAddItem()
            return
        }

        if (event.key === 'Tab' && field === 'name' && !event.shiftKey) {
            event.preventDefault()
            expandToPrice()
            return
        }

        if (event.key === 'Tab' && field === 'price_per_unit' && !event.shiftKey) {
            event.preventDefault()
            quantityRef.current?.focus()
            return
        }

        if (event.key === 'Tab' && field === 'quantity' && event.shiftKey) {
            event.preventDefault()
            priceRef.current?.focus()
            return
        }

        if (event.key === 'Tab' && field === 'price_per_unit' && event.shiftKey) {
            event.preventDefault()
            nameRef.current?.focus()
            return
        }

        if (event.key === 'Backspace' && field === 'name') {
            const value = (event.target as HTMLInputElement).value
            if (value === '' && rowCount > 1) {
                event.preventDefault()
                onRemove()
            }
            return
        }

        if (event.key === 'ArrowDown' && index < rowCount - 1) {
            event.preventDefault()
            onFocusField(index + 1, field)
            return
        }

        if (event.key === 'ArrowUp' && index > 0) {
            event.preventDefault()
            onFocusField(index - 1, field)
        }
    }, [expandToPrice, index, onAddItem, onFocusField, onRemove, rowCount])

    const formattedTotal = formatCurrency(lineTotal, currency, { showSymbol: false })
    const foldLabel = expanded ? t('actions.collapse') : t('actions.expand')

    return (
        <div className="min-w-0 rounded-lg border px-2 py-1.5">
            <div className="flex min-w-0 items-start gap-1">
                <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    className="size-6 shrink-0 text-muted-foreground"
                    onClick={() => {
                        if (expanded) {
                            onToggle()
                            return
                        }
                        pendingPriceFocus.current = true
                        onExpand()
                    }}
                    aria-expanded={expanded}
                    aria-label={foldLabel}
                    title={foldLabel}
                >
                    <ChevronDown className={cn('size-3 transition-transform duration-200', expanded && 'rotate-180')} />
                </Button>

                <FormField
                    control={form.control}
                    name={`items.${index}.name`}
                    render={({ field }) => (
                        <FormItem className="min-w-0 flex-1 space-y-1">
                            <div className="flex min-w-0 items-center gap-2">
                                <FormControl>
                                    <Input
                                        {...field}
                                        ref={(element) => {
                                            field.ref(element)
                                            nameRef.current = element
                                        }}
                                        placeholder={t('forms:transactions.itemName')}
                                        className="h-8 min-w-0 border-0 shadow-none focus-visible:ring-1"
                                        data-item-name
                                        data-index={index}
                                        onFocus={() => onExpand()}
                                        onKeyDown={(event) => handleItemKeyDown(event, 'name')}
                                    />
                                </FormControl>
                            </div>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <div className="flex h-8 w-16 shrink-0 items-center justify-end font-mono text-sm text-muted-foreground tabular-nums">
                    {formattedTotal}
                </div>

                <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    onClick={onRemove}
                    disabled={!canRemove}
                    className="shrink-0 text-muted-foreground hover:text-destructive"
                    aria-label={t('actions.delete')}
                    title={t('actions.delete')}
                >
                    <Trash2 className="size-4" />
                </Button>
            </div>

            <div
                className={cn(
                    'grid min-w-0 transition-[grid-template-rows,opacity] duration-200 ease-out',
                    expanded ? 'grid-rows-[1fr] opacity-100' : 'grid-rows-[0fr] opacity-0',
                )}
                inert={!expanded}
            >
                <div className="min-h-0 overflow-hidden">
                    <div className="grid min-w-0 grid-cols-[minmax(0,7rem)_minmax(0,1fr)] items-start gap-3 py-1 pl-7">
                        <FormField
                            control={form.control}
                            name={`items.${index}.quantity`}
                            render={({ field }) => (
                                <FormItem className="min-w-0 space-y-1">
                                    <div className="flex min-w-0 items-center gap-2">
                                        <FormLabel className="m-0 w-16 shrink-0 text-xs font-normal text-muted-foreground">
                                            {t('forms:transactions.qty')}
                                        </FormLabel>
                                        <FormControl>
                                            <Input
                                                {...field}
                                                ref={(element) => {
                                                    field.ref(element)
                                                    quantityRef.current = element
                                                }}
                                                type="number"
                                                step={priceInputStep(ITEM_QTY_DECIMALS)}
                                                min={0}
                                                placeholder="1"
                                                className="h-8 min-w-0"
                                                data-item-quantity
                                                data-index={index}
                                                value={field.value ?? ''}
                                                onFocus={() => onExpand()}
                                                onChange={(event) => {
                                                    field.onChange(
                                                        event.target.value === ''
                                                            ? ''
                                                            : Number(event.target.value),
                                                    )
                                                }}
                                                onKeyDown={(event) => handleItemKeyDown(event, 'quantity')}
                                            />
                                        </FormControl>
                                    </div>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name={`items.${index}.price_per_unit`}
                            render={({ field }) => (
                                <FormItem className="min-w-0 space-y-1">
                                    <div className="flex min-w-0 items-center gap-2">
                                        <FormLabel className="m-0 w-16 shrink-0 text-xs font-normal text-muted-foreground">
                                            {t('forms:transactions.price')}
                                        </FormLabel>
                                        <FormControl>
                                            <Input
                                                {...field}
                                                ref={(element) => {
                                                    field.ref(element)
                                                    priceRef.current = element
                                                }}
                                                type="number"
                                                step={priceInputStep(decimals)}
                                                min={0}
                                                placeholder={decimals <= 0 ? '0' : (0).toFixed(decimals)}
                                                className="h-8 min-w-0"
                                                data-item-price_per_unit
                                                data-index={index}
                                                value={field.value ?? ''}
                                                onFocus={() => onExpand()}
                                                onChange={(event) => {
                                                    field.onChange(
                                                        event.target.value === ''
                                                            ? ''
                                                            : Number(event.target.value),
                                                    )
                                                }}
                                                onKeyDown={(event) => handleItemKeyDown(event, 'price_per_unit')}
                                            />
                                        </FormControl>
                                    </div>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                    </div>
                </div>
            </div>
        </div>
    )
}
