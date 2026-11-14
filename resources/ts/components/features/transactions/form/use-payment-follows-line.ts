import { useEffect, useRef } from 'react'
import { useFormContext, useWatch } from 'react-hook-form'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Account } from '@/types'

const blank = (value: unknown): boolean => value === undefined || value === null || String(value).trim() === ''

/** Whether the line is the whole transaction: one payment, one line, no change, and the line in the payment's currency. */
function isWhole(values: Partial<TransactionFormValues>, accounts?: Account[]): boolean {
    const { lines, payments, change } = values
    if (lines?.length !== 1 || payments?.length !== 1 || (change?.length ?? 0) > 0) {
        return false
    }
    const currency = accounts?.find((a) => a.id === Number(payments[0]?.account_id))?.currency
    const lineCurrency = lines[0]?.currency_id
    return !lineCurrency || !currency || Number(lineCurrency) === currency.id
}

/**
 * When the form opens with an amount on one side only (an opener gave the payment's), the other side takes it. After
 * that the payment and the lines are tied only by the lock of the payment, see `useUnlockedPayments`: with it closed
 * the payment is the user's and the lines never change with it.
 */
export function useStartingAmount(accounts?: Account[]): void {
    const { control, setValue } = useFormContext<TransactionFormValues>()
    const [payments, change, lines] = useWatch({ control, name: ['payments', 'change', 'lines'] })
    const lineAmount = lines?.[0]?.amount
    const payAmount = payments?.[0]?.amount
    const whole = isWhole({ payments, change, lines }, accounts)

    // Only the first time: later, a cleared amount is the user's.
    const done = useRef(false)
    useEffect(() => {
        if (!whole || done.current) {
            return
        }
        done.current = true
        if (blank(lineAmount) !== blank(payAmount)) {
            if (blank(lineAmount)) {
                setValue('lines.0.amount', payAmount as number)
            } else {
                setValue('payments.0.amount', lineAmount as number)
            }
        }
    }, [whole, lineAmount, payAmount, setValue])
}
