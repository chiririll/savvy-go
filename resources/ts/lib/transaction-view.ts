import type { Payment, Transaction } from '@/types'

/**
 * Whether a payment is change: money handed back, the opposite sign of what the transaction does.
 * Only an expense or an income has change; in a transfer both signs are real payments.
 */
export function isChangePayment(transaction: Pick<Transaction, 'type'>, payment: Pick<Payment, 'amount'>): boolean {
    if (transaction.type === 'expense') {
        return payment.amount > 0
    }
    if (transaction.type === 'income') {
        return payment.amount < 0
    }
    return false
}

/** The lines of a transaction in their saved order. */
export function orderedLines<T extends { seq: number }>(lines: T[]): T[] {
    return [...lines].sort((a, b) => a.seq - b.seq)
}
