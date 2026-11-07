import type { TransactionFormValues } from '@/schemas/transactions'
import type { Transaction } from '@/types'
import { roundToCurrency } from './transaction-payload'

/**
 * Whether the form can show a saved transaction without losing anything: an expense or an income paid
 * with one payment of the sign its type has, in one line for all of it, in the currency of the account.
 */
export function isSimpleTransaction(transaction: Transaction): boolean {
    if (transaction.type !== 'expense' && transaction.type !== 'income') {
        return false
    }
    const [payment, ...otherPayments] = transaction.payments
    const [line, ...otherLines] = transaction.lines
    const currency = payment?.account?.currency
    if (!payment || otherPayments.length > 0 || !line || otherLines.length > 0 || !currency) {
        return false
    }
    const sign = transaction.type === 'income' ? 1 : -1
    return (
        Math.sign(payment.amount) === sign &&
        line.currencyId === currency.id &&
        roundToCurrency(line.amount, currency) === roundToCurrency(Math.abs(payment.amount), currency)
    )
}

/**
 * Whether the form can edit a saved transaction without losing anything: an expense or an income with one
 * payment of the sign its type has, and lines, all in the currency of that account. Several payments and
 * transfers come with the later steps of the form.
 */
export function canEditInForm(transaction: Transaction): boolean {
    if (transaction.type !== 'expense' && transaction.type !== 'income') {
        return false
    }
    const [payment, ...otherPayments] = transaction.payments
    const currency = payment?.account?.currency
    if (!payment || otherPayments.length > 0 || !currency || transaction.lines.length === 0) {
        return false
    }
    const sign = transaction.type === 'income' ? 1 : -1
    return Math.sign(payment.amount) === sign && transaction.lines.every((line) => line.currencyId === currency.id)
}

/** The form values for a transaction `canEditInForm` accepts. */
export function toFormValues(transaction: Transaction): TransactionFormValues {
    const [payment] = transaction.payments
    const [line] = transaction.lines

    // One line for all of it is the simple mode: its category is the category of the transaction.
    if (!isSimpleTransaction(transaction)) {
        return {
            type: transaction.type as 'income' | 'expense',
            description: transaction.description ?? '',
            date: transaction.date ?? '',
            tag_ids: transaction.tags.map((tag) => tag.id),
            payments: [{ account_id: payment.accountId, amount: Math.abs(payment.amount) }],
            change: [],
            lines: [...transaction.lines]
                .sort((a, b) => a.seq - b.seq)
                .map((l) => ({ category_id: l.categoryId, currency_id: l.currencyId, amount: l.amount, note: l.note })),
            default_category_id: null,
            default_note: null,
        }
    }

    return {
        type: transaction.type as 'income' | 'expense',
        description: transaction.description ?? '',
        date: transaction.date ?? '',
        tag_ids: transaction.tags.map((tag) => tag.id),
        payments: [{ account_id: payment.accountId, amount: Math.abs(payment.amount) }],
        change: [],
        lines: [],
        default_category_id: line.categoryId,
        default_note: line.note,
    }
}
