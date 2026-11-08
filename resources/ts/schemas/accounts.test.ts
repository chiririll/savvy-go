import { describe, expect, it } from 'vitest'

import { accountSchema, toAccountPayload } from './accounts'

const bank = { name: 'Main', type: 'bank', currency: 'id:1' }
const card = { name: 'Visa', type: 'credit', currency: 'id:1', limit: 2000, statement_day: 28, grace_days: 21 }

describe('accountSchema', () => {
    it('does not ask a bank account for card terms', () => {
        expect(accountSchema.safeParse(bank).success).toBe(true)
    })

    it('accepts a card with its terms', () => {
        expect(accountSchema.safeParse(card).success).toBe(true)
    })

    it.each([
        ['no limit', { limit: undefined }, 'limit'],
        ['zero limit', { limit: 0 }, 'limit'],
        ['day 0', { statement_day: 0 }, 'statement_day'],
        ['day 32', { statement_day: 32 }, 'statement_day'],
        ['no day', { statement_day: undefined }, 'statement_day'],
        ['grace over 60', { grace_days: 61 }, 'grace_days'],
        ['negative grace', { grace_days: -1 }, 'grace_days'],
    ])('refuses a card with %s', (_name, override, field) => {
        const parsed = accountSchema.safeParse({ ...card, ...override })
        expect(parsed.success).toBe(false)
        expect(parsed.error?.issues.map((issue) => issue.path[0])).toContain(field)
    })

    it('accepts a decimal limit typed as text', () => {
        expect(accountSchema.safeParse({ ...card, limit: '1500.50' }).success).toBe(true)
    })
})

describe('toAccountPayload', () => {
    it('drops the card terms of a bank account', () => {
        const data = accountSchema.parse({ ...bank, limit: 5, statement_day: 1, grace_days: 1 })
        expect(toAccountPayload(data)).toEqual({
            name: 'Main', type: 'bank', currency_id: 1, initial_balance: 0, is_active: true,
        })
    })

    it('keeps the card terms of a card', () => {
        expect(toAccountPayload(accountSchema.parse(card))).toMatchObject({
            type: 'credit', limit: 2000, statement_day: 28, grace_days: 21, currency_id: 1,
        })
    })
})
