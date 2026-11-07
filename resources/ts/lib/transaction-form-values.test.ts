import { describe, expect, it } from 'vitest'

import { accounts, line, payment, tags, transaction } from '@/test/fixtures'
import { isSimpleTransaction, toFormValues } from './transaction-form-values'

const [cash, euroCard] = accounts

describe('isSimpleTransaction', () => {
    it('is true for one payment and one line that covers it', () => {
        expect(isSimpleTransaction(transaction())).toBe(true)
        expect(
            isSimpleTransaction(transaction({ type: 'income', payments: [payment({ amount: 42.5 })] })),
        ).toBe(true)
    })

    it('is false for a transfer, a debt payment or anything else the form cannot show', () => {
        expect(isSimpleTransaction(transaction({ type: 'transfer' }))).toBe(false)
        expect(isSimpleTransaction(transaction({ type: 'debt_payment' }))).toBe(false)
    })

    it('is false with several payments, change included', () => {
        const paid = payment({ amount: -50 })
        const back = payment({ account: euroCard, accountId: euroCard.id, amount: 5 })
        expect(isSimpleTransaction(transaction({ payments: [paid, back] }))).toBe(false)
    })

    it('is false with several lines, or none', () => {
        expect(isSimpleTransaction(transaction({ lines: [line(), line({ seq: 1 })] }))).toBe(false)
        expect(isSimpleTransaction(transaction({ lines: [] }))).toBe(false)
    })

    it('is false when the line is not for all of the payment', () => {
        expect(isSimpleTransaction(transaction({ lines: [line({ amount: 40 })] }))).toBe(false)
        expect(isSimpleTransaction(transaction({ lines: [line({ currencyId: 2 })] }))).toBe(false)
    })

    it('is false when the payment has the wrong sign for its type', () => {
        expect(isSimpleTransaction(transaction({ payments: [payment({ amount: 42.5 })] }))).toBe(false)
    })
})

describe('toFormValues', () => {
    it('shows an expense with a positive amount and the category of its line', () => {
        const tx = transaction({ tags: [tags[0]], description: 'Weekly shop' })

        expect(toFormValues(tx)).toEqual({
            type: 'expense',
            description: 'Weekly shop',
            date: '2020-05-04',
            tag_ids: [tags[0].id],
            payments: [{ account_id: cash.id, amount: 42.5 }],
            change: [],
            lines: [],
            default_category_id: 1,
            default_note: null,
        })
    })

    it('keeps the note of the line, so editing does not lose it', () => {
        const tx = transaction({ lines: [line({ note: '- milk\n- bread' })] })

        expect(toFormValues(tx).default_note).toBe('- milk\n- bread')
    })

    it('shows an income with a positive amount', () => {
        const tx = transaction({ type: 'income', payments: [payment({ amount: 42.5 })] })

        expect(toFormValues(tx).payments).toEqual([{ account_id: cash.id, amount: 42.5 }])
    })

    it('shows a pending transaction without a date as an empty one', () => {
        expect(toFormValues(transaction({ date: null })).date).toBe('')
        expect(toFormValues(transaction({ description: null })).description).toBe('')
    })
})
