import type { Ref } from 'react'
import type { UseFormReturn } from 'react-hook-form'

import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import type { TransactionFormValues } from '@/schemas/transactions'

interface ItemNumberFieldProps {
    form: UseFormReturn<TransactionFormValues>
    index: number
    name: 'quantity' | 'price_per_unit'
    label: string
    placeholder: string
    inputRef: { current: HTMLInputElement | null }
    onFocus: () => void
    onKeyDown: (event: React.KeyboardEvent<HTMLInputElement>) => void
}

/** A labelled number input inside an expanded item row; empty text stays '' instead of becoming 0. */
export function ItemNumberField({
    form,
    index,
    name,
    label,
    placeholder,
    inputRef,
    onFocus,
    onKeyDown,
}: ItemNumberFieldProps) {
    return (
        <FormField
            control={form.control}
            name={`items.${index}.${name}`}
            render={({ field }) => (
                <FormItem className="min-w-0 space-y-1">
                    <div className="flex min-w-0 items-center gap-2">
                        <FormLabel className="m-0 w-16 shrink-0 text-xs font-normal text-muted-foreground">
                            {label}
                        </FormLabel>
                        <FormControl>
                            <Input
                                {...field}
                                ref={
                                    ((element: HTMLInputElement | null) => {
                                        field.ref(element)
                                        inputRef.current = element
                                    }) as Ref<HTMLInputElement>
                                }
                                type="number"
                                min={0}
                                placeholder={placeholder}
                                className="h-8 min-w-0 flex-1"
                                {...{ [`data-item-${name}`]: true }}
                                data-index={index}
                                value={field.value ?? ''}
                                onFocus={onFocus}
                                onChange={(event) =>
                                    field.onChange(event.target.value === '' ? '' : Number(event.target.value))
                                }
                                onKeyDown={onKeyDown}
                            />
                        </FormControl>
                    </div>
                    <FormMessage />
                </FormItem>
            )}
        />
    )
}
