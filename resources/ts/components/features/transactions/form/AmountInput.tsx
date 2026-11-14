import type { ComponentProps } from 'react'
import { sanitizeAmount } from '@/lib/format/amount-input'
import { Input } from '@/components/ui/input'
import type { Currency } from '@/types'

type AmountInputProps = Omit<ComponentProps<typeof Input>, 'type' | 'value' | 'onChange'> & {
    value?: string | number | null
    onChange: (value: string) => void
    /** The currency of the amount: it sets the decimals and is shown at the end of the field. */
    currency?: Currency | null
    /** Keep the decimals of the currency but do not draw its code: a currency picker sits beside the field. */
    hideCode?: boolean
}

/** An amount of money: plain text that takes digits and one decimal separator, no more digits than the currency has. */
export function AmountInput({ value, onChange, currency, hideCode, className, onFocus, ...props }: AmountInputProps) {
    const decimals = currency?.decimals ?? 2

    return (
        <div className="relative">
            <Input
                type="text"
                inputMode="decimal"
                autoComplete="off"
                placeholder={decimals > 0 ? `0.${'0'.repeat(decimals)}` : '0'}
                className={currency && !hideCode ? `pr-12 ${className ?? ''}` : className}
                {...props}
                value={value ?? ''}
                onChange={(event) => onChange(sanitizeAmount(event.target.value, decimals))}
                onFocus={(event) => {
                    event.currentTarget.select()
                    onFocus?.(event)
                }}
            />
            {currency && !hideCode && (
                <span className="pointer-events-none absolute inset-y-0 right-3 flex items-center text-xs text-muted-foreground">
                    {currency.code}
                </span>
            )}
        </div>
    )
}
