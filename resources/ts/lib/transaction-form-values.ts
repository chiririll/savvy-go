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

/** The form values for a simple transaction (see `isSimpleTransaction`). */
export function toFormValues(transaction: Transaction): TransactionFormValues {
    const [payment] = transaction.payments
    const [line] = transaction.lines

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
