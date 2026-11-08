import { describe, expect, it } from 'vitest'
import { getTransactionSchema, type TransactionFormValues } from './transactions'

const base: TransactionFormValues = {
    type: 'expense',
    description: 'Groceries',
    date: '2020-10-17',
    tag_ids: [],
    payments: [{ account_id: 1, amount: 450 }],
    change: [],
    lines: [],
    default_category_id: 7,
    default_note: null,
}

const issues = (values: Partial<TransactionFormValues>, options?: Parameters<typeof getTransactionSchema>[0]) => {
    const result = getTransactionSchema(options).safeParse({ ...base, ...values })
    return result.success ? [] : result.error.issues.map((issue) => issue.path.join('.'))
}

describe('the transaction form', () => {
    it('accepts one payment with the default category', () => {
        expect(issues({})).toEqual([])
    })

    it('accepts an income', () => {
        expect(issues({ type: 'income' })).toEqual([])
    })

    it('needs an account and a positive amount in a payment', () => {
        expect(issues({ payments: [{ account_id: 0, amount: 0 }] })).toEqual([
            'payments.0.account_id',
            'payments.0.amount',
        ])
    })

    it('needs at least one payment', () => {
        expect(issues({ payments: [] })).toEqual(['payments'])
    })

    it('needs a category when there are no lines', () => {
        expect(issues({ default_category_id: null })).toEqual(['default_category_id'])
    })

    it('lets the default category stay empty when lines are written', () => {
        const lines = [{ category_id: 3, amount: 450, currency_id: 1, note: null }]
        expect(issues({ default_category_id: null, lines })).toEqual([])
    })

    it('needs a positive amount in a line', () => {
        const lines = [{ category_id: 3, amount: 0, currency_id: 1, note: null }]
        expect(issues({ lines })).toEqual(['lines.0.amount'])
    })

    it('refuses the same account in two payments, at the second', () => {
        const payments = [
            { account_id: 1, amount: 5 },
            { account_id: 1, amount: 5 },
        ]
        expect(issues({ payments })).toEqual(['payments.1.account_id'])
    })

    it('refuses an account that is both paid from and given the change', () => {
        const payments = [
            { account_id: 1, amount: 5 },
            { account_id: 2, amount: 5 },
        ]
        expect(issues({ payments, change: [{ account_id: 2, amount: 1 }] })).toEqual(['change.0.account_id'])
    })

    it('does not count an account not chosen yet as a duplicate', () => {
        const payments = [
            { account_id: 0, amount: 5 },
            { account_id: 0, amount: 5 },
        ]
        expect(issues({ payments })).toEqual(['payments.0.account_id', 'payments.1.account_id'])
    })

    it('takes a description of one line only', () => {
        expect(issues({ description: 'a\nb' })).toEqual(['description'])
    })

    it('lets the date be empty, which makes it pending', () => {
        expect(issues({ date: '' })).toEqual([])
        expect(issues({ date: null })).toEqual([])
    })

    it('refuses a future date for a transaction that already moved balances', () => {
        expect(issues({ date: '2999-01-01' }, { rejectFutureDate: true })).toEqual(['date'])
        expect(issues({ date: '2999-01-01' })).toEqual([])
    })

    describe('a transfer', () => {
        const transfer: Partial<TransactionFormValues> = {
            type: 'transfer',
            default_category_id: null,
            to_account_id: 2,
            to_amount: 80,
        }

        it('needs no category', () => {
            expect(issues(transfer)).toEqual([])
        })

        it('needs a destination account, and an amount to receive', () => {
            expect(issues({ ...transfer, to_account_id: null, to_amount: null })).toEqual(['to_account_id', 'to_amount'])
        })

        it('does not send to the account it takes from', () => {
            expect(issues({ ...transfer, to_account_id: 1 })).toEqual(['to_account_id'])
        })

        it('accepts a fee with an amount, in no category or in one', () => {
            expect(issues({ ...transfer, fee: { amount: 1.5, category_id: null } })).toEqual([])
            expect(issues({ ...transfer, fee: { amount: 1.5, category_id: 3 } })).toEqual([])
        })

        it('needs the fee to have an amount once it is asked for', () => {
            expect(issues({ ...transfer, fee: { amount: 0, category_id: null } })).toEqual(['fee.amount'])
        })

        it('still needs a source account and an amount', () => {
            expect(issues({ ...transfer, payments: [{ account_id: 0, amount: 0 }] })).toEqual([
                'payments.0.account_id',
                'payments.0.amount',
            ])
        })

        it('to a linked space also needs the space; the account may have any id there', () => {
            const out = { ...transfer, type: 'transfer_out' as const, to_account_id: 1 }

            expect(issues(out)).toEqual(['to_space_id'])
            expect(issues({ ...out, to_space_id: 5 })).toEqual([])
        })
    })
})
