import { describe, expect, it, vi } from 'vitest'

import type { Transaction } from '@/types/transactions'
import {
    displayTransactionDescription,
    transactionAmountAppearance,
    transactionSubtitle,
} from './transaction-description'

// Echoes keys so the tests see which message was picked.
vi.mock('@/lib/i18n', () => ({
    default: {
        t: (key: string, options?: { name?: string }) => (options?.name ? `${key}(${options.name})` : key),
    },
}))

const account = { name: 'Cash' }
const base = { account, category: null, toAccount: null, description: null }
type Input = Parameters<typeof displayTransactionDescription>[0]
const tx = (overrides: Partial<Record<keyof Input, unknown>>) => ({ ...base, type: 'expense', ...overrides }) as unknown as Input

describe('displayTransactionDescription', () => {
    it('prefers the stored description', () => {
        expect(displayTransactionDescription(tx({ description: '  Lunch  ' }))).toBe('Lunch')
    })

    it('ignores a stored message key and builds a fallback', () => {
        expect(displayTransactionDescription(tx({ description: 'messages.debt_payment', type: 'debt_payment' })))
            .toBe('pages:transactions.fallback.payment(Cash)')
    })

    it('ignores a blank description', () => {
        expect(displayTransactionDescription(tx({ description: '   ' }))).toBe('pages:transactions.types.expense')
    })

    it('shows both accounts of a transfer', () => {
        expect(displayTransactionDescription(tx({ type: 'transfer', toAccount: { name: 'Bank' } }))).toBe('Cash → Bank')
        expect(displayTransactionDescription(tx({ type: 'transfer' }))).toBe('Cash')
    })

    it.each([
        ['debt_payment', 'payment'],
        ['debt_collection', 'collection'],
        ['debt_lend', 'lend'],
        ['debt_borrow', 'borrow'],
    ])('describes %s by the debt account', (type, key) => {
        expect(displayTransactionDescription(tx({ type, toAccount: { name: 'Debt' } })))
            .toBe(`pages:transactions.fallback.${key}(Debt)`)
        expect(displayTransactionDescription(tx({ type })))
            .toBe(`pages:transactions.fallback.${key}(Cash)`)
    })

    it('falls back to the category, then to the type', () => {
        expect(displayTransactionDescription(tx({ category: { name: 'Taxi' } }))).toBe('Taxi')
        expect(displayTransactionDescription(tx({ type: 'income' }))).toBe('pages:transactions.types.income')
    })
})

describe('transactionSubtitle', () => {
    it('joins the account and the category', () => {
        expect(transactionSubtitle({ account, category: { name: 'Taxi' } } as unknown as Transaction)).toBe('Cash · Taxi')
    })

    it('shows only the account without a category', () => {
        expect(transactionSubtitle({ account, category: null } as unknown as Transaction)).toBe('Cash')
    })
})

describe('transactionAmountAppearance', () => {
    it.each(['income', 'debt_collection', 'debt_borrow', 'transfer_in'] as const)('%s is a green plus', (type) => {
        expect(transactionAmountAppearance(type)).toEqual({ sign: '+', className: 'text-green-600' })
    })

    it.each(['expense', 'debt_payment', 'debt_lend', 'transfer_out'] as const)('%s is a red minus', (type) => {
        expect(transactionAmountAppearance(type)).toEqual({ sign: '-', className: 'text-red-600' })
    })

    it('shows a transfer between own accounts without a sign', () => {
        expect(transactionAmountAppearance('transfer')).toEqual({ sign: '', className: 'text-blue-600' })
    })

    it('mutes skipped transactions whatever the type', () => {
        expect(transactionAmountAppearance('income', 'skipped')).toEqual({ sign: '', className: 'text-muted-foreground' })
        expect(transactionAmountAppearance('expense', 'skipped')).toEqual({ sign: '', className: 'text-muted-foreground' })
    })

    it('does not mute other statuses', () => {
        expect(transactionAmountAppearance('income', 'pending').sign).toBe('+')
        expect(transactionAmountAppearance('income', 'confirmed').sign).toBe('+')
    })
})
