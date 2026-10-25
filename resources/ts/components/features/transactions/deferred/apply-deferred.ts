import type { ConfirmTransactionOptions } from '@/api'

export type ApplyDateChoice = 'today' | 'original' | 'other'
export type ApplyOptions = ConfirmTransactionOptions & { date: string }

export function selectedApplyDate(
    choice: ApplyDateChoice,
    today: string,
    originalDate: string | null,
    customDate: string,
): string | null {
    if (choice === 'today') return today
    if (choice === 'original') return originalDate
    return customDate
}

/** The options to confirm with; the amounts are sent only for an estimate, the second only across currencies. */
export function buildApplyOptions(input: {
    date: string
    estimated: boolean
    crossCurrency: boolean
    amount: number
    toAmount: number
}): ApplyOptions {
    const options: ApplyOptions = { date: input.date }
    if (input.estimated) {
        options.amount = input.amount
        if (input.crossCurrency) options.to_amount = input.toAmount
    }
    return options
}

/** An estimate needs real amounts before it can be confirmed. */
export const amountsValid = (estimated: boolean, crossCurrency: boolean, amount: number, toAmount: number) =>
    !estimated || (amount > 0 && (!crossCurrency || toAmount > 0))
