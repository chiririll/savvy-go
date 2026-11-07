import type { TransactionFormValues } from '@/schemas/transactions'
import type { Payment, Transaction, TransactionLine } from '@/types'
import { paymentSign, roundToCurrency } from './transaction-payload'

const signOf = (transaction: Transaction): 1 | -1 => paymentSign(transaction.type === 'income' ? 'income' : 'expense')

/** The payments that take or bring money as the type does, and the change: the ones of the opposite sign. */
function splitPayments(transaction: Transaction): { paid: Payment[]; back: Payment[] } {
    const sign = signOf(transaction)
    return {
        paid: transaction.payments.filter((p) => Math.sign(p.amount) === sign),
        back: transaction.payments.filter((p) => Math.sign(p.amount) === -sign),
    }
}

/**
 * Whether the form shows a saved transaction in the simple mode, with the category of its only line as the
 * category of the transaction: an expense or an income, one line, and every payment (change included) in the
 * currency of that line, netting to the amount of the line.
 */
export function isSimpleTransaction(transaction: Transaction): boolean {
    if (transaction.type !== 'expense' && transaction.type !== 'income') {
        return false
    }
    const [line, ...otherLines] = transaction.lines
    const { paid } = splitPayments(transaction)
    if (!line || otherLines.length > 0 || paid.length === 0) {
        return false
    }
    const currency = paid[0].account?.currency
    if (!currency || line.currencyId !== currency.id) {
        return false
    }
    if (transaction.payments.some((p) => p.account?.currency?.id !== currency.id)) {
        return false
    }
    const net = transaction.payments.reduce((sum, p) => sum + p.amount, 0)
    return roundToCurrency(line.amount, currency) === roundToCurrency(Math.abs(net), currency)
}

/**
 * The parts of a saved transfer the form shows: the payment that takes money, the one that brings it, and the fee
 * (the only line, in the currency of the source). Null when it is anything else, such as the sides of a transfer to
 * another space, a transfer with more accounts or lines, or one whose fee is all it took.
 */
function transferParts(transaction: Transaction): { from: Payment; to: Payment; fee: TransactionLine | null } | null {
    if (transaction.type !== 'transfer' || transaction.payments.length !== 2 || transaction.lines.length > 1) {
        return null
    }
    const from = transaction.payments.find((p) => p.amount < 0)
    const to = transaction.payments.find((p) => p.amount > 0)
    if (!from || !to || !from.account?.currency || !to.account?.currency) {
        return null
    }
    const fee = transaction.lines[0] ?? null
    if (fee && (fee.currencyId !== from.account.currency.id || !(Math.abs(from.amount) > fee.amount))) {
        return null
    }
    return { from, to, fee }
}

/**
 * Whether the form can edit a saved transaction without losing anything. An expense or an income with at least
 * one payment of the sign its type has, the others being change (the opposite sign), none of them zero, and
 * lines in the currency of one of the payment accounts. Or a transfer between two accounts of this space with at
 * most a fee, that the backend does not restrict.
 */
export function canEditInForm(transaction: Transaction): boolean {
    if (transaction.restrictions.length > 0) {
        return false
    }
    if (transaction.type === 'transfer') {
        return transferParts(transaction) !== null
    }
    if (transaction.type !== 'expense' && transaction.type !== 'income') {
        return false
    }
    const { paid, back } = splitPayments(transaction)
    if (paid.length === 0 || paid.length + back.length !== transaction.payments.length) {
        return false
    }
    const currencies = transaction.payments.map((p) => p.account?.currency?.id)
    if (currencies.some((id) => id === undefined) || transaction.lines.length === 0) {
        return false
    }
    return transaction.lines.every((line) => currencies.includes(line.currencyId))
}

function transferFormValues(transaction: Transaction, parts: NonNullable<ReturnType<typeof transferParts>>): TransactionFormValues {
    const { from, to, fee } = parts
    const currency = from.account!.currency!
    return {
        type: 'transfer',
        description: transaction.description ?? '',
        date: transaction.date ?? '',
        tag_ids: transaction.tags.map((tag) => tag.id),
        // The fee is paid on top of what is sent.
        payments: [{ account_id: from.accountId, amount: roundToCurrency(Math.abs(from.amount) - (fee?.amount ?? 0), currency) }],
        change: [],
        lines: [],
        default_category_id: null,
        default_note: null,
        to_account_id: to.accountId,
        to_amount: to.amount,
        to_space_id: null,
        fee: fee ? { amount: fee.amount, category_id: fee.categoryId } : null,
    }
}

/** The form values for a transaction `canEditInForm` accepts. */
export function toFormValues(transaction: Transaction): TransactionFormValues {
    const parts = transferParts(transaction)
    if (parts) {
        return transferFormValues(transaction, parts)
    }
    const { paid, back } = splitPayments(transaction)
    const row = (p: Payment) => ({ account_id: p.accountId, amount: Math.abs(p.amount) })
    const common = {
        type: transaction.type as 'income' | 'expense',
        description: transaction.description ?? '',
        date: transaction.date ?? '',
        tag_ids: transaction.tags.map((tag) => tag.id),
        payments: paid.map(row),
        change: back.map(row),
    }

    // One line for all of it is the simple mode: its category is the category of the transaction.
    if (isSimpleTransaction(transaction)) {
        const [line] = transaction.lines
        return { ...common, lines: [], default_category_id: line.categoryId, default_note: line.note }
    }

    return {
        ...common,
        lines: [...transaction.lines]
            .sort((a, b) => a.seq - b.seq)
            .map((l) => ({ category_id: l.categoryId, currency_id: l.currencyId, amount: l.amount, note: l.note })),
        default_category_id: null,
        default_note: null,
    }
}
