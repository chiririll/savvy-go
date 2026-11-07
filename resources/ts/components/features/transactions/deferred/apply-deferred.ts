import type { ConfirmTransactionOptions } from '@/api'
import { collectNegativeBalanceWarnings, warningIfResultNegative, type NegativeBalanceWarning } from '@/lib/negative-balance'
import { roundToCurrency } from '@/lib/transaction-payload'
import type { LineInput, PaymentInput, Transaction, TransactionLine } from '@/types'

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

/**
 * The amounts the user types when an occurrence is confirmed, as text and without sign: one per payment in the
 * order of the transaction, and one per line in the order of their sequence.
 */
export interface AmountDraft {
    payments: string[]
    lines: string[]
}

/** The open occurrence of a recurring rule: the only pending transaction whose amounts can be corrected. */
export const isOccurrence = (transaction: Transaction): boolean =>
    transaction.restrictions.some((r) => r.reason === 'recurring_pending')

const linesInOrder = (transaction: Transaction): TransactionLine[] =>
    [...transaction.lines].sort((a, b) => a.seq - b.seq)

/**
 * Whether the only line is the payments' net, in the one currency they share: its amount is not typed, it follows
 * the payments, as the form's simple mode does.
 */
export function linesFollowPayments(transaction: Transaction): boolean {
    const [line, ...others] = transaction.lines
    if (!line || others.length > 0 || transaction.payments.length === 0) {
        return false
    }
    return transaction.payments.every((p) => p.account?.currency?.id === line.currencyId)
}

export function initialAmounts(transaction: Transaction): AmountDraft {
    return {
        payments: transaction.payments.map((p) => String(Math.abs(p.amount))),
        lines: linesInOrder(transaction).map((l) => String(l.amount)),
    }
}

const same = (typed: string, amount: number) => typed.trim() !== '' && Number(typed) === amount

/** Whether the typed amounts differ from the ones the occurrence has; if not, they are not sent. */
export function amountsChanged(transaction: Transaction, draft: AmountDraft): boolean {
    if (transaction.payments.some((p, i) => !same(draft.payments[i] ?? '', Math.abs(p.amount)))) {
        return true
    }
    if (linesFollowPayments(transaction)) {
        return false
    }
    return linesInOrder(transaction).some((l, i) => !same(draft.lines[i] ?? '', l.amount))
}

const positive = (typed: string | undefined) => Number(typed) > 0 && (typed ?? '').trim() !== ''

/** An occurrence needs a real amount for every payment, and for every line that is typed. */
export function amountsValid(transaction: Transaction, draft: AmountDraft): boolean {
    if (!isOccurrence(transaction)) {
        return true
    }
    if (transaction.payments.some((_, i) => !positive(draft.payments[i]))) {
        return false
    }
    if (linesFollowPayments(transaction)) {
        return true
    }
    return transaction.lines.every((_, i) => positive(draft.lines[i]))
}

/** The payments as they would be confirmed: the typed amounts, with the sign each payment already has. */
function typedPayments(transaction: Transaction, draft: AmountDraft): PaymentInput[] {
    return transaction.payments.map((p, i) => {
        const typed = Number(draft.payments[i])
        const amount = p.account?.currency ? roundToCurrency(typed, p.account.currency) : typed
        return { account_id: p.accountId, amount: Math.sign(p.amount) * amount }
    })
}

function typedLines(transaction: Transaction, draft: AmountDraft, payments: PaymentInput[]): LineInput[] {
    const follow = linesFollowPayments(transaction)
    return linesInOrder(transaction).map((l, i) => {
        const currency = transaction.payments.find((p) => p.account?.currency?.id === l.currencyId)?.account?.currency
        let amount = Number(draft.lines[i])
        if (follow) {
            // What the payments net to, in the currency they share.
            const net = Math.abs(payments.reduce((sum, p) => sum + (p.amount ?? 0), 0))
            amount = currency ? roundToCurrency(net, currency) : net
        } else if (currency) {
            amount = roundToCurrency(amount, currency)
        }
        return { category_id: l.categoryId, currency_id: l.currencyId, amount, note: l.note }
    })
}

/**
 * The options to confirm with. The amounts are sent only for an occurrence whose amounts the user changed: payments
 * with their sign, and the lines too when the transaction has some, since the server keeps the two apart.
 */
export function buildApplyOptions(transaction: Transaction, date: string, draft: AmountDraft): ApplyOptions {
    const options: ApplyOptions = { date }
    if (!isOccurrence(transaction) || !amountsChanged(transaction, draft)) {
        return options
    }
    options.payments = typedPayments(transaction, draft)
    if (transaction.lines.length > 0) {
        options.lines = typedLines(transaction, draft, options.payments)
    }
    return options
}

/** The accounts that confirming leaves below zero: the payments as they will be posted, money going out only. */
export function applyWarnings(transaction: Transaction, draft: AmountDraft): NegativeBalanceWarning[] {
    const payments =
        isOccurrence(transaction) && amountsValid(transaction, draft)
            ? typedPayments(transaction, draft)
            : transaction.payments.map((p) => ({ account_id: p.accountId, amount: p.amount }))

    return collectNegativeBalanceWarnings(
        transaction.payments.map((p, i) => {
            const delta = payments[i]?.amount ?? p.amount
            if (!p.account || !(delta < 0)) {
                return null
            }
            return warningIfResultNegative({
                accountName: p.account.name,
                currentBalance: p.account.currentBalance,
                resultingBalance: roundTo(p.account.currentBalance + delta, p.account.currency?.decimals ?? 2),
                currency: p.account.currency,
            })
        }),
    )
}

const roundTo = (value: number, decimals: number) => {
    const factor = 10 ** decimals
    return Math.round(value * factor) / factor
}
