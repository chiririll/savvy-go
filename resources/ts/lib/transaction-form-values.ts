import type { TransactionFormValues } from '@/schemas/transactions'
import type { Payment, Transaction } from '@/types'
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
 * Whether the form can edit a saved transaction without losing anything: an expense or an income with at least
 * one payment of the sign its type has, the others being change (the opposite sign), none of them zero, and
 * lines in the currency of one of the payment accounts. Transfers come with the later steps of the form.
 */
export function canEditInForm(transaction: Transaction): boolean {
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

/** The form values for a transaction `canEditInForm` accepts. */
export function toFormValues(transaction: Transaction): TransactionFormValues {
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
