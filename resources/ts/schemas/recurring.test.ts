import { describe, expect, it } from 'vitest'
import { recurringSchema, type RecurringFormValues } from './recurring'

const base: RecurringFormValues = {
    type: 'expense',
    description: 'Rent',
    date: null,
    tag_ids: [],
    payments: [{ account_id: 1, amount: 1100 }],
    change: [],
    lines: [],
    default_category_id: 7,
    default_note: null,
    estimated: false,
    frequency: 'monthly',
    interval: 1,
    day_of_week: 1,
    day_of_month: 5,
    start_date: '2026-12-05',
    end_date: null,
    is_active: true,
}

const issues = (values: Partial<RecurringFormValues>) => {
    const result = recurringSchema.safeParse({ ...base, ...values })
    return result.success ? [] : result.error.issues.map((issue) => issue.path.join('.'))
}

describe('the recurring form', () => {
    it('accepts a monthly expense with a category', () => {
        expect(issues({})).toEqual([])
    })

    it('has no date to ask for', () => {
        expect(issues({ date: null })).toEqual([])
    })

    it('refuses a type a rule cannot make', () => {
        expect(issues({ type: 'transfer_out' as RecurringFormValues['type'] })).toEqual(['type'])
    })

    it('needs a category when there are no lines, as a transaction does', () => {
        expect(issues({ default_category_id: null })).toEqual(['default_category_id'])
    })

    it('refuses an account used twice', () => {
        expect(
            issues({ payments: [{ account_id: 1, amount: 5 }], change: [{ account_id: 1, amount: 1 }] }),
        ).toEqual(['change.0.account_id'])
    })

    it('needs the amount of a payment: only a transfer may leave one open', () => {
        expect(issues({ payments: [{ account_id: 1, amount: 0 }] })).toEqual(['payments.0.amount'])
    })

    it('lets a transfer leave what it receives open', () => {
        expect(
            issues({ type: 'transfer', default_category_id: null, to_account_id: 2, to_amount: null }),
        ).toEqual([])
    })

    it('still needs the destination of a transfer', () => {
        expect(issues({ type: 'transfer', default_category_id: null, to_account_id: null })).toEqual(['to_account_id'])
    })

    it('refuses a transfer to its own account', () => {
        expect(issues({ type: 'transfer', default_category_id: null, to_account_id: 1 })).toEqual(['to_account_id'])
    })

    it('needs an interval of at least one', () => {
        expect(issues({ interval: 0 })).toEqual(['interval'])
    })

    it('needs a start', () => {
        expect(issues({ start_date: '' })).toEqual(['start_date'])
    })

    it('refuses an end before the start', () => {
        expect(issues({ end_date: '2026-01-01' })).toEqual(['end_date'])
    })

    it('accepts an end on the start', () => {
        expect(issues({ end_date: '2026-12-05' })).toEqual([])
    })
})
