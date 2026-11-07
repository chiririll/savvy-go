import { describe, expect, it } from 'vitest'

import { accounts, payment, transaction } from '@/test/fixtures'
import { formImpact, postedImpact, projectBalances } from './balance-preview'

const [cash, euroCard] = accounts

describe('formImpact', () => {
    it('takes an expense off the account and brings an income to it', () => {
        expect(formImpact({ type: 'expense', payments: [{ account_id: 1, amount: 10 }] })).toEqual([
            { accountId: 1, delta: -10 },
        ])
        expect(formImpact({ type: 'income', payments: [{ account_id: 1, amount: 10 }] })).toEqual([
            { accountId: 1, delta: 10 },
        ])
    })

    it('gives the change the opposite sign', () => {
        const impact = formImpact({
            type: 'expense',
            payments: [{ account_id: 1, amount: 10 }],
            change: [{ account_id: 2, amount: 3 }],
        })

        expect(impact).toEqual([
            { accountId: 1, delta: -10 },
            { accountId: 2, delta: 3 },
        ])
    })

    it('skips rows with no account or amount yet', () => {
        expect(formImpact({ type: 'expense', payments: [{ account_id: 0, amount: 10 }] })).toEqual([])
        expect(formImpact({ type: 'expense', payments: [{ account_id: 1, amount: Number.NaN }] })).toEqual([])
    })
})

describe('postedImpact', () => {
    it('is what the saved payments did, signed as stored', () => {
        const tx = transaction({ payments: [payment({ amount: -10 }), payment({ account: euroCard, accountId: 2, amount: 3 })] })

        expect(postedImpact(tx)).toEqual([
            { accountId: 1, delta: -10 },
            { accountId: 2, delta: 3 },
        ])
    })
})

describe('projectBalances', () => {
    it('shows the balance before and after for each account of the payments', () => {
        const rows = projectBalances(accounts, [{ accountId: cash.id, delta: -20 }], null)

        expect(rows).toEqual([
            {
                account: cash,
                currentBalance: 120.5,
                newBalance: 100.5,
                insufficientFunds: false,
                currency: cash.currency,
            },
        ])
    })

    it('takes back what the saved transaction already did', () => {
        // The balance already has -10 in it; the edit makes it -25: 120.50 + 10 - 25
        const rows = projectBalances(accounts, [{ accountId: cash.id, delta: -25 }], [{ accountId: cash.id, delta: -10 }])

        expect(rows[0].newBalance).toBe(105.5)
    })

    it('adds in whole minor units, with no float noise', () => {
        const rows = projectBalances(
            [{ ...cash, currentBalance: 0.3 }],
            [{ accountId: cash.id, delta: -0.1 }],
            null,
        )

        expect(rows[0].newBalance).toBe(0.2)
    })

    it('flags an account the expense takes below zero', () => {
        const rows = projectBalances(accounts, [{ accountId: cash.id, delta: -200 }], null)

        expect(rows[0]).toMatchObject({ newBalance: -79.5, insufficientFunds: true })
    })

    it('does not flag an income, nor an account that was already below zero and gains', () => {
        const low = { ...cash, currentBalance: -5 }
        expect(projectBalances([low], [{ accountId: low.id, delta: 2 }], null)[0].insufficientFunds).toBe(false)
    })

    it('leaves out an account that is not loaded', () => {
        expect(projectBalances(accounts, [{ accountId: 99, delta: -1 }], null)).toEqual([])
    })

    it('lists a changed account that the saved transaction used and the edit no longer does', () => {
        const rows = projectBalances(accounts, [{ accountId: euroCard.id, delta: -4 }], [{ accountId: cash.id, delta: -10 }])

        expect(rows.map((r) => r.account.id)).toEqual([euroCard.id])
    })
})
