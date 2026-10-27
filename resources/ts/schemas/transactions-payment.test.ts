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
    it('is an ordinary expense when paid from one account', () => {
        expect(issues({ payment_mode: 'single' })).toEqual([])
        expect(issues({})).toEqual([])
    })

    it('needs the other accounts that paid, each with an account and an amount', () => {
        expect(issues({ payment_mode: 'several' })).toEqual(['extra_payments'])
        expect(issues({ payment_mode: 'several', extra_payments: [{ account_id: 0, amount: 0 }] })).toEqual([
            'extra_payments.0.account_id',
            'extra_payments.0.amount',
        ])
        expect(issues({ payment_mode: 'several', extra_payments: [{ account_id: 3, amount: 5 }] })).toEqual([])
    })

    it('with change needs what was handed over, above the price, and where the change went', () => {
        expect(issues({ payment_mode: 'change' }).sort()).toEqual(['change_account_id', 'change_amount', 'handed_over'])
        expect(
            issues({ payment_mode: 'change', handed_over: 450, change_account_id: 2, change_amount: 10 }),
        ).toEqual(['handed_over'])
        expect(
            issues({ payment_mode: 'change', handed_over: 500, change_account_id: 2, change_amount: 5900 }),
        ).toEqual([])
    })

    it('is settled: it needs a date that is not in the future', () => {
        const paid = { payment_mode: 'change', handed_over: 500, change_account_id: 2, change_amount: 5900 } as const
        expect(issues({ ...paid, date: null })).toEqual(['date'])
        expect(issues({ ...paid, date: '2999-01-01' })).toEqual(['date'])
    })
})
