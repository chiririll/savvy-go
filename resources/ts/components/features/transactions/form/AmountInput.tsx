import { useEffect, useState, type ComponentProps, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { evaluateAmount, isFormula, sanitizeAmountInput } from '@/lib/format/amount-formula'
import { Input } from '@/components/ui/input'
import type { Currency } from '@/types'

type AmountInputProps = Omit<ComponentProps<typeof Input>, 'type' | 'value' | 'onChange'> & {
    value?: string | number | null
    onChange: (value: string) => void
    /** The currency of the amount: it sets the decimals and is shown at the end of the field. */
    currency?: Currency | null
    /** Keep the decimals of the currency but do not draw its code: a currency picker sits beside the field. */
    hideCode?: boolean
    /** At the end of the field, after the currency code: a button that is part of the field, such as the lock. */
    end?: ReactNode
}

/**
 * An amount of money: plain text that takes digits and one decimal separator, no more digits than the currency has.
 * It also takes a formula (`12,5*3 + (40-5)/2`), which becomes its result when the field loses focus; one that does
 * not work stays as typed and marks the field. A read-only field neither takes nor evaluates anything.
 */
export function AmountInput({ value, onChange, currency, hideCode, end, className, onFocus, onBlur, readOnly, ...props }: AmountInputProps) {
    const { t } = useTranslation('common')
    const decimals = currency?.decimals ?? 2
    const [broken, setBroken] = useState(false)
    // A formula is held here while it is typed: the form only hears of a number, or of a formula that failed.
    const [draft, setDraft] = useState<string | null>(null)
    useEffect(() => setDraft(null), [value])

    const commit = (text: string) => {
        const result = evaluateAmount(text, decimals)
        if (result.kind === 'value') {
            setDraft(null)
            onChange(result.value)
        } else if (result.kind === 'error') {
            onChange(text)
        }
        setBroken(result.kind === 'error')
        return result.kind
    }
    const showCode = currency && !hideCode
    const padding = end ? (showCode ? 'pr-20' : 'pr-10') : showCode ? 'pr-12' : ''

    return (
        <div className="relative">
            <Input
                type="text"
                inputMode="decimal"
                autoComplete="off"
                placeholder={decimals > 0 ? `0.${'0'.repeat(decimals)}` : '0'}
                className={`${padding} ${readOnly ? 'bg-muted/40 text-muted-foreground' : ''} ${className ?? ''}`.trim()}
                {...props}
                readOnly={readOnly}
                aria-invalid={broken || props['aria-invalid']}
                title={broken ? t('validation.amountFormula') : props.title}
                value={draft ?? value ?? ''}
                onChange={(event) => {
                    setBroken(false)
                    const text = sanitizeAmountInput(event.target.value, decimals)
                    if (isFormula(text)) {
                        setDraft(text)
                    } else {
                        setDraft(null)
                        onChange(text)
                    }
                }}
                onKeyDown={(event) => {
                    // Enter works a formula out; the form does not submit on it, see `noEnterSubmit`.
                    if (event.key === 'Enter' && !readOnly && isFormula(event.currentTarget.value)) {
                        event.preventDefault()
                        commit(event.currentTarget.value)
                    }
                    props.onKeyDown?.(event)
                }}
                onBlur={(event) => {
                    if (!readOnly) {
                        commit(event.currentTarget.value)
                    }
                    onBlur?.(event)
                }}
                onFocus={(event) => {
                    event.currentTarget.select()
                    onFocus?.(event)
                }}
            />
            {(showCode || end) && (
                <div className="pointer-events-none absolute inset-y-0 right-1 flex items-center gap-1">
                    {showCode && <span className="pr-2 text-xs text-muted-foreground">{currency.code}</span>}
                    {end && <span className="pointer-events-auto">{end}</span>}
                </div>
            )}
        </div>
    )
}
