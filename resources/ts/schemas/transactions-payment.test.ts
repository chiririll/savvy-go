import { describe, expect, it } from 'vitest'
import { getTransactionSchema, type TransactionFormValues } from './transactions'

const base: TransactionFormValues = {
    type: 'expense',
    account_id: 1,
    category_id: 7,
    amount: 450,
    date: '2020-10-17',
}

const issues = (values: Partial<TransactionFormValues>) => {
    const result = getTransactionSchema().safeParse({ ...base, ...values })
    return result.success ? [] : result.error.issues.map((issue) => issue.path.join('.'))
}

describe('a payment in several parts', () => {
    it('is an ordinary expense with no rows besides the first', () => {
        expect(issues({})).toEqual([])
        expect(issues({ extra_payments: [], changes: [] })).toEqual([])
    })

    it('needs an account and an amount in each extra row', () => {
        expect(issues({ extra_payments: [{ account_id: 0, amount: 0 }] })).toEqual([
            'extra_payments.0.account_id',
            'extra_payments.0.amount',
        ])
        expect(issues({ extra_payments: [{ account_id: 3, amount: 5 }] })).toEqual([])
    })

    it('needs an account and an amount in each change row', () => {
        expect(issues({ changes: [{ account_id: 0, amount: 0 }] })).toEqual([
            'changes.0.account_id',
            'changes.0.amount',
        ])
        expect(issues({ changes: [{ account_id: 2, amount: 5900 }] })).toEqual([])
    })

    it('is settled: it needs a date that is not in the future', () => {
        const paid = { changes: [{ account_id: 2, amount: 5900 }] }
        expect(issues({ ...paid, date: null })).toEqual(['date'])
        expect(issues({ ...paid, date: '2999-01-01' })).toEqual(['date'])
    })
})
