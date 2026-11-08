import { useEffect, useRef } from 'react'
import { useFormContext, useWatch, type UseFormReturn } from 'react-hook-form'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { Account } from '@/types'

const blank = (value: unknown): boolean => value === undefined || value === null || String(value).trim() === ''
const same = (a: unknown, b: unknown): boolean => (blank(a) && blank(b)) || (!blank(a) && !blank(b) && Number(a) === Number(b))

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
 * With one payment and one line, the line is the whole transaction: editing the amount of either one updates the
 * other. Call it from the change handler of the amount that is edited, before the edit is stored: it sets the other
 * amount then, so both fields change in one render. (Doing it in a form subscription instead notifies the fields
 * from inside a notification, and the ones notified last are left with the old value.) A line that covers less than
 * its payment (a saved one) is left alone until the user makes them equal.
 */
export function followAmount(
    form: Pick<UseFormReturn<TransactionFormValues>, 'getValues' | 'setValue'>,
    accounts: Account[] | undefined,
    edited: 'line' | 'payment',
    value: unknown,
): void {
    const values = form.getValues()
    if (!isWhole(values, accounts) || !same(values.lines?.[0]?.amount, values.payments?.[0]?.amount)) {
        return
    }
    if (edited === 'line') {
        form.setValue('payments.0.amount', value as number)
    } else {
        form.setValue('lines.0.amount', value as number)
    }
}

/** When the form opens with an amount on one side only (an opener gave the payment's), the other side takes it. */
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
