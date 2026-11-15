import { describe, expect, it } from 'vitest'
import { accounts, categories, line, payment, tags, transaction } from '@/test/fixtures'
import type { Transaction, TransactionActions } from '@/types'
import {
    buildDetailsInput,
    formModeOf,
    refusalReason,
    toDetailsValues,
    type DetailsFormValues,
} from './transaction-details'

const actions = (overrides: Partial<TransactionActions>): TransactionActions => ({
    edit: false,
    editDetails: false,
    duplicate: false,
    delete: false,
    confirm: false,
    skip: false,
    ...overrides,
})

const restricted = (reason: Transaction['restrictions'][number]['reason'], allows: Partial<TransactionActions>) =>
    transaction({ restrictions: [{ reason, message: reason }], actions: actions(allows) })

describe('formModeOf', () => {
    it('opens a free transaction in the whole form', () => {
        expect(formModeOf(transaction())).toBe('full')
    })

    it('opens a transaction the whole form cannot show in the restricted one, when details may change', () => {
        const odd = transaction({ type: 'transfer_out', payments: [payment({ amount: -30 })], lines: [] })
        expect(formModeOf(odd)).toBe('details')
    })

    it.each([
        ['debt', { editDetails: true, delete: true }],
        ['space_transfer', { edit: true, editDetails: true }],
    ] as const)('opens a transaction restricted by %s in the restricted form', (reason, allows) => {
        expect(formModeOf(restricted(reason, allows))).toBe('details')
    })

    it('refuses what nothing can change', () => {
        expect(formModeOf(restricted('transfer_review', {}))).toBeNull()
        expect(formModeOf(restricted('recurring_pending', { confirm: true, skip: true }))).toBeNull()
        expect(formModeOf(transaction({ actions: actions({ delete: true }) }))).toBeNull()
    })

    it('opens a transaction that allows only its details in the restricted form', () => {
        expect(formModeOf(transaction({ actions: actions({ editDetails: true }) }))).toBe('details')
    })
})

describe('refusalReason', () => {
    it('names the restriction that blocks every change', () => {
        expect(refusalReason(restricted('transfer_review', {}))).toBe('transfer_review')
        expect(refusalReason(restricted('recurring_pending', { confirm: true }))).toBe('recurring_pending')
    })

    it('has none for a transaction without restrictions', () => {
        expect(refusalReason(transaction({ actions: actions({}) }))).toBeNull()
    })
})

describe('toDetailsValues', () => {
    it('takes the description, the tags and the category and note of each line in order', () => {
        const tx = transaction({
            description: 'Trip',
            tags: [tags[0]],
            lines: [
                line({ seq: 1, categoryId: categories[1].id, note: '- b' }),
                line({ seq: 0, categoryId: categories[0].id, note: null }),
            ],
        })
        expect(toDetailsValues(tx)).toEqual({
            description: 'Trip',
            tag_ids: [tags[0].id],
            lines: [
                { category_id: categories[0].id, note: null },
                { category_id: categories[1].id, note: '- b' },
            ],
        })
    })

    it('has an empty description for none', () => {
        expect(toDetailsValues(transaction({ description: null })).description).toBe('')
    })
})

describe('buildDetailsInput', () => {
    const tx = transaction({
        payments: [payment({ amount: -30 }), payment({ account: accounts[1], accountId: 2, amount: -10 })],
        lines: [
            line({ seq: 0, amount: 30, currencyId: 1 }),
            line({ seq: 1, amount: 10, currencyId: 2, categoryId: categories[1].id, note: 'old' }),
        ],
    })
    const values: DetailsFormValues = {
        description: ' New title ',
        tag_ids: [3],
        lines: [
            { category_id: categories[0].id, note: '' },
            { category_id: null, note: '  - milk  ' },
        ],
    }

    it('sends the description, the tags and the lines with their own amounts and currencies, no payments', () => {
        const input = buildDetailsInput(tx, values)
        expect(input).toEqual({
            description: 'New title',
            tag_ids: [3],
            lines: [
                { category_id: categories[0].id, currency_id: 1, amount: 30, note: null },
                { category_id: null, currency_id: 2, amount: 10, note: '- milk' },
            ],
        })
        expect(input).not.toHaveProperty('payments')
        expect(input).not.toHaveProperty('type')
        expect(input).not.toHaveProperty('date')
    })

    it('sends the tags even when none are left, so they are cleared', () => {
        expect(buildDetailsInput(tx, { ...values, tag_ids: [] }).tag_ids).toEqual([])
    })

    it('leaves the lines out of a transaction that has none', () => {
        const debt = transaction({ type: 'debt_payment', lines: [] })
        const input = buildDetailsInput(debt, { description: 'x', tag_ids: [], lines: [] })
        expect(input).toEqual({ description: 'x', tag_ids: [] })
    })

    it('clears the description when it is blank', () => {
        expect(buildDetailsInput(tx, { ...values, description: '   ' }).description).toBe('')
    })
})
