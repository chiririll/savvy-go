import { ChevronDown, Trash2 } from 'lucide-react'
import { useFormContext, type UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { FormControl, FormField, FormItem, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { cn, formatCurrency } from '@/lib/utils'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Currency } from '@/types'

import { ItemNumberField } from './ItemNumberField'
import type { ItemField } from './item-keys'
import { useItemRowFocus } from './use-item-row-focus'

interface TransactionItemRowProps {
    index: number
    form: UseFormReturn<TransactionFormValues>
    currency?: Currency | null
    decimals: number
    quantity: number
    lineTotal: number
    expanded: boolean
    onExpand: () => void
    onToggle: () => void
    canRemove: boolean
    onRemove: () => void
    onAddItem: () => void
    onFocusField: (index: number, field: ItemField) => void
    rowCount: number
}

export function TransactionItemRow({
    index,
    form,
    currency,
    decimals,
    quantity,
    lineTotal,
    expanded,
    onExpand,
    onToggle,
    canRemove,
    onRemove,
    onAddItem,
    onFocusField,
    rowCount,
}: TransactionItemRowProps) {
    const { t } = useTranslation(['common', 'forms'])
    const { getFieldState, formState } = useFormContext<TransactionFormValues>()
    const quantityState = getFieldState(`items.${index}.quantity`, formState)
    const priceState = getFieldState(`items.${index}.price_per_unit`, formState)
    const { nameRef, quantityRef, priceRef, handleKeyDown, handleFoldClick } = useItemRowFocus({
        index,
        rowCount,
        expanded,
        quantityError: quantityState.error,
        priceError: priceState.error,
        onExpand,
        onToggle,
        onRemove,
        onAddItem,
        onFocusField,
    })

    const formattedTotal = formatCurrency(lineTotal, currency, { showSymbol: false })
    const formattedQuantity = Number(quantity.toFixed(2))
    const showQuantity = !expanded && quantity > 0 && formattedQuantity !== 1
    const foldLabel = expanded ? t('actions.collapse') : t('actions.expand')

    return (
        <div className="min-w-0 rounded-lg border px-2 py-1.5">
            <div className="flex min-w-0 items-start gap-1">
                <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    className="size-6 shrink-0 text-muted-foreground"
                    onClick={handleFoldClick}
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
                                        onKeyDown={(event) => handleKeyDown(event, 'name')}
                                    />
                                </FormControl>
                            </div>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <div
                    className={cn(
                        'flex h-8 min-w-16 shrink-0 items-center justify-end gap-1 font-mono text-sm',
                        'text-muted-foreground tabular-nums',
                    )}
                >
                    {showQuantity && <span className="text-xs">×{formattedQuantity} =</span>}
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
                    <div
                        className={cn(
                            'grid min-w-0 grid-cols-1 sm:grid-cols-[minmax(0,12rem)_minmax(0,1fr)]',
                            'items-start gap-3 py-1 pl-7',
                        )}
                    >
                        <ItemNumberField
                            form={form}
                            index={index}
                            name="quantity"
                            label={t('forms:transactions.qty')}
                            placeholder="1"
                            inputRef={quantityRef}
                            onFocus={onExpand}
                            onKeyDown={(event) => handleKeyDown(event, 'quantity')}
                        />
                        <ItemNumberField
                            form={form}
                            index={index}
                            name="price_per_unit"
                            label={t('forms:transactions.price')}
                            placeholder={decimals <= 0 ? '0' : (0).toFixed(decimals)}
                            inputRef={priceRef}
                            onFocus={onExpand}
                            onKeyDown={(event) => handleKeyDown(event, 'price_per_unit')}
                        />
                    </div>
                </div>
            </div>
        </div>
    )
}
