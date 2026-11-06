import { describe, expect, it, vi } from 'vitest'

import {
    displayTransactionDescription,
    transactionAmountAppearance,
    transactionSubtitle,
} from './transaction-description'

// Echoes keys so the tests see which message was picked; SALARY is the only translated default name.
vi.mock('@/lib/i18n', () => ({
    default: {
        exists: (key: string) => key === 'categories.SALARY',
        t: (key: string, options?: { name?: string }) => (options?.name ? `${key}(${options.name})` : key),
    },
}))

const acc = (id: number, name: string, type = 'bank') => ({ id, name, type, currency: { id: 1 } })
const pay = (id: number, name: string, amount: number, type?: string) => ({
    account: acc(id, name, type),
    accountId: id,
    amount,
})
const ln = (id: number | null, name = '') => ({ categoryId: id, category: id ? { id, name } : null, currencyId: 1, amount: 1 })
const tx = (overrides: Record<string, unknown> = {}) =>
    ({
        type: 'expense',
        description: null,
        payments: [pay(1, 'Cash', -5)],
        lines: [ln(null)],
        ...overrides,
    }) as unknown as Input
type Input = Parameters<typeof displayTransactionDescription>[0]

describe('displayTransactionDescription', () => {
    it('prefers the stored description', () => {
        expect(displayTransactionDescription(tx({ description: '  Lunch  ' }))).toBe('Lunch')
    })

    it('ignores a stored message key and builds a fallback', () => {
        expect(displayTransactionDescription(tx({ description: 'messages.debt_payment', type: 'debt_payment' }))).toBe(
            'pages:transactions.fallback.payment(Cash)',
        )
    })

    it('ignores a blank description', () => {
        expect(displayTransactionDescription(tx({ description: '   ' }))).toBe('pages:transactions.types.expense')
    })

    it('shows both accounts of a transfer, by the sign of the payments', () => {
        const payments = [pay(2, 'Bank', 5), pay(1, 'Cash', -5)]
        expect(displayTransactionDescription(tx({ type: 'transfer', payments }))).toBe('Cash → Bank')
        expect(displayTransactionDescription(tx({ type: 'transfer' }))).toBe('Cash')
    })

    it.each([
        ['debt_payment', 'payment'],
        ['debt_collection', 'collection'],
        ['debt_lend', 'lend'],
        ['debt_borrow', 'borrow'],
    ])('describes %s by the debt account', (type, key) => {
        const payments = [pay(1, 'Cash', -5), pay(9, 'Debt', 5, 'debt')]
        expect(displayTransactionDescription(tx({ type, payments }))).toBe(`pages:transactions.fallback.${key}(Debt)`)
        expect(displayTransactionDescription(tx({ type }))).toBe(`pages:transactions.fallback.${key}(Cash)`)
    })

    it('falls back to the category, then to the type', () => {
        expect(displayTransactionDescription(tx({ lines: [ln(1, '#SALARY')] }))).toBe('categories.SALARY')
        expect(displayTransactionDescription(tx({ lines: [ln(1, 'Taxi')] }))).toBe('Taxi')
        expect(displayTransactionDescription(tx({ type: 'income' }))).toBe('pages:transactions.types.income')
    })
})

describe('transactionSubtitle', () => {
    it('joins the account and the category', () => {
        expect(transactionSubtitle(tx({ lines: [ln(1, 'Taxi')] }))).toBe('Cash · Taxi')
        expect(transactionSubtitle(tx({ lines: [ln(1, '#SALARY')] }))).toBe('Cash · categories.SALARY')
    })

    it('shows only the account without a category', () => {
        expect(transactionSubtitle(tx())).toBe('Cash')
    })

    it('counts the categories of several lines, never shows a note', () => {
        const lines = [ln(1, 'Food'), ln(2, 'Home'), ln(2, 'Home')]
        expect(transactionSubtitle(tx({ lines }))).toBe('Cash · pages:transactions.categoriesCount')
    })

    it('shows the first account and how many others paid', () => {
        const payments = [pay(1, 'Cash', -5), pay(2, 'Card', -5), pay(2, 'Card', -1), pay(3, 'Bank', 3)]
        expect(transactionSubtitle(tx({ payments }))).toBe('Cash +2')
    })

    it('leaves the debt account out while an ordinary one paid', () => {
        const payments = [pay(9, 'Debt', 5, 'debt'), pay(1, 'Cash', -5)]
        expect(transactionSubtitle(tx({ payments }))).toBe('Cash')
    })
})

describe('transactionAmountAppearance', () => {
    it.each(['income', 'debt_collection', 'debt_borrow', 'transfer_in'] as const)('%s is a green plus', (type) => {
        expect(transactionAmountAppearance(type)).toEqual({ sign: '+', className: 'text-success' })
    })

    it.each(['expense', 'debt_payment', 'debt_lend', 'transfer_out'] as const)('%s is a red minus', (type) => {
        expect(transactionAmountAppearance(type)).toEqual({ sign: '-', className: 'text-destructive' })
    })

    it('shows a transfer between own accounts without a sign', () => {
        expect(transactionAmountAppearance('transfer')).toEqual({ sign: '', className: 'text-info' })
    })

    it('mutes skipped transactions whatever the type', () => {
        expect(transactionAmountAppearance('income', 'skipped')).toEqual({
            sign: '',
            className: 'text-muted-foreground',
        })
        expect(transactionAmountAppearance('expense', 'skipped')).toEqual({
            sign: '',
            className: 'text-muted-foreground',
        })
    })

    it('does not mute other statuses', () => {
        expect(transactionAmountAppearance('income', 'pending').sign).toBe('+')
        expect(transactionAmountAppearance('income', 'confirmed').sign).toBe('+')
    })
})
