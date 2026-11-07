import { describe, expect, it } from 'vitest'

import { accounts, line, payment, tags, transaction } from '@/test/fixtures'
import { canEditInForm, isSimpleTransaction, toFormValues } from './transaction-form-values'

const [cash, euroCard, yen] = accounts
const paidFrom = (account: (typeof accounts)[number], amount: number) =>
    payment({ account, accountId: account.id, amount })

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

    it('is false with change in another currency than the line', () => {
        const paid = payment({ amount: -50 })
        const back = paidFrom(euroCard, 5)
        expect(isSimpleTransaction(transaction({ payments: [paid, back] }))).toBe(false)
    })

    it('is true for payments in the currency of a line that covers their net', () => {
        const other = { ...cash, id: 9, name: 'Wallet' }
        const paid = [payment({ amount: -30 }), paidFrom(other, -15), paidFrom({ ...cash, id: 10 }, 2.5)]
        expect(isSimpleTransaction(transaction({ payments: paid }))).toBe(true)
    })

    it('is false with several payments when the line does not cover what they net to', () => {
        const other = { ...cash, id: 9, name: 'Wallet' }
        expect(isSimpleTransaction(transaction({ payments: [payment({ amount: -30 }), paidFrom(other, -15)] }))).toBe(false)
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

describe('canEditInForm', () => {
    it('is true for several payments, with change in any currency', () => {
        const other = { ...cash, id: 9, name: 'Wallet' }
        expect(canEditInForm(transaction({ payments: [payment({ amount: -30 }), paidFrom(other, -15)] }))).toBe(true)
        expect(canEditInForm(transaction({ payments: [payment({ amount: -50 }), paidFrom(euroCard, 5)] }))).toBe(true)
        expect(canEditInForm(transaction({ payments: [payment({ amount: -50 }), paidFrom(yen, 700)] }))).toBe(true)
    })

    it('is true for an income paid into several accounts', () => {
        expect(
            canEditInForm(
                transaction({ type: 'income', payments: [paidFrom(cash, 30), paidFrom(euroCard, 8)], lines: [line({ amount: 40 })] }),
            ),
        ).toBe(true)
    })

    it('is true for a line in the currency of a change account', () => {
        const payments = [payment({ amount: -50 }), paidFrom(euroCard, 5)]
        expect(canEditInForm(transaction({ payments, lines: [line({ currencyId: euroCard.currency!.id, amount: 4 })] }))).toBe(true)
    })

    it('is true for a simple transaction and for one payment split in several lines', () => {
        expect(canEditInForm(transaction())).toBe(true)
        expect(canEditInForm(transaction({ lines: [line({ amount: 40 }), line({ seq: 1, amount: 2.5 })] }))).toBe(true)
    })

    it('is true for lines that cover less than the payment', () => {
        expect(canEditInForm(transaction({ lines: [line({ amount: 40 })] }))).toBe(true)
    })

    it('is false for a transfer, a debt payment or no lines', () => {
        expect(canEditInForm(transaction({ type: 'transfer' }))).toBe(false)
        expect(canEditInForm(transaction({ type: 'debt_payment' }))).toBe(false)
        expect(canEditInForm(transaction({ lines: [] }))).toBe(false)
    })

    it('is false with no payment of the sign of the type', () => {
        expect(canEditInForm(transaction({ payments: [payment({ amount: 42.5 })] }))).toBe(false)
        expect(canEditInForm(transaction({ payments: [] }))).toBe(false)
    })

    it('is false when a payment is zero, which the form cannot show', () => {
        expect(canEditInForm(transaction({ payments: [payment({ amount: -50 }), paidFrom(euroCard, 0)] }))).toBe(false)
    })

    it('is false when a line is in a currency none of the payment accounts has', () => {
        expect(canEditInForm(transaction({ lines: [line({ currencyId: 2 })] }))).toBe(false)
        expect(canEditInForm(transaction({ payments: [payment({ amount: -50 }), paidFrom(euroCard, 5)], lines: [line({ currencyId: 3 })] }))).toBe(false)
    })
})

describe('toFormValues', () => {
    it('shows several lines as lines, with the category, amount and note of each, in their order', () => {
        const tx = transaction({
            lines: [
                line({ seq: 1, categoryId: 3, category: null, amount: 2.5, note: null }),
                line({ seq: 0, categoryId: 1, amount: 40, note: '- milk\n- bread' }),
            ],
        })

        expect(toFormValues(tx)).toMatchObject({
            payments: [{ account_id: cash.id, amount: 42.5 }],
            lines: [
                { category_id: 1, currency_id: 1, amount: 40, note: '- milk\n- bread' },
                { category_id: 3, currency_id: 1, amount: 2.5, note: null },
            ],
            default_category_id: null,
            default_note: null,
        })
    })

    it('shows the payments and the change apart, with positive amounts', () => {
        const tx = transaction({
            payments: [payment({ amount: -50 }), paidFrom(euroCard, -8), paidFrom(yen, 700)],
            lines: [line({ amount: 50 }), line({ seq: 1, amount: 10 })],
        })

        const values = toFormValues(tx)
        expect(values.payments).toEqual([
            { account_id: cash.id, amount: 50 },
            { account_id: euroCard.id, amount: 8 },
        ])
        expect(values.change).toEqual([{ account_id: yen.id, amount: 700 }])
    })

    it('shows an income with change as payments and change the same way', () => {
        const tx = transaction({
            type: 'income',
            payments: [paidFrom(cash, 30), paidFrom(euroCard, -2)],
            lines: [line({ amount: 28 })],
        })

        expect(toFormValues(tx)).toMatchObject({
            payments: [{ account_id: cash.id, amount: 30 }],
            change: [{ account_id: euroCard.id, amount: 2 }],
        })
    })

    it('shows several payments with one line that covers them as the simple mode', () => {
        const other = { ...cash, id: 9, name: 'Wallet' }
        const tx = transaction({ payments: [payment({ amount: -30 }), paidFrom(other, -12.5)] })

        expect(toFormValues(tx)).toMatchObject({
            payments: [
                { account_id: cash.id, amount: 30 },
                { account_id: 9, amount: 12.5 },
            ],
            change: [],
            lines: [],
            default_category_id: 1,
        })
    })

    it('shows a line that covers less as a line, not as the category', () => {
        const values = toFormValues(transaction({ lines: [line({ amount: 40 })] }))

        expect(values.lines).toHaveLength(1)
        expect(values.default_category_id).toBeNull()
    })

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
