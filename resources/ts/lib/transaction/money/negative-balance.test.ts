import { describe, expect, it } from 'vitest'

import {
    collectNegativeBalanceWarnings,
    warningIfResultNegative,
    warningsForAccountOutflow,
    warningsForConfirmedDuplicate,
} from './negative-balance'

const account = { name: 'Cash', currentBalance: 100 }

describe('warningIfResultNegative', () => {
    it('warns when a spend takes the balance below zero', () => {
        expect(
            warningIfResultNegative({
                accountName: 'Cash',
                currentBalance: 100,
                resultingBalance: -20,
                currency: 'USD',
            }),
        ).toEqual({ accountName: 'Cash', currentBalance: 100, resultingBalance: -20, currency: 'USD' })
    })

    it('does not warn when the balance stays at or above zero', () => {
        expect(warningIfResultNegative({ accountName: 'Cash', currentBalance: 100, resultingBalance: 0 })).toBeNull()
        expect(warningIfResultNegative({ accountName: 'Cash', currentBalance: 100, resultingBalance: 50 })).toBeNull()
    })

    it('does not warn when the balance did not go down', () => {
        expect(warningIfResultNegative({ accountName: 'Cash', currentBalance: -50, resultingBalance: -50 })).toBeNull()
        expect(warningIfResultNegative({ accountName: 'Cash', currentBalance: -50, resultingBalance: -10 })).toBeNull()
    })

    it('warns again when an already negative balance gets worse', () => {
        expect(
            warningIfResultNegative({ accountName: 'Cash', currentBalance: -50, resultingBalance: -60 }),
        ).not.toBeNull()
    })

    it('needs an account name', () => {
        expect(warningIfResultNegative({ accountName: '', currentBalance: 10, resultingBalance: -10 })).toBeNull()
    })
})

describe('collectNegativeBalanceWarnings', () => {
    it('drops empty candidates', () => {
        const w = warningIfResultNegative({ accountName: 'Cash', currentBalance: 1, resultingBalance: -1 })!
        expect(collectNegativeBalanceWarnings([null, w, undefined])).toEqual([w])
    })
})

describe('warningsForAccountOutflow', () => {
    it('subtracts the amount from the current balance', () => {
        expect(warningsForAccountOutflow(account, 100)).toEqual([])
        expect(warningsForAccountOutflow(account, 100.01)).toHaveLength(1)
    })

    it('ignores a missing account and non-positive amounts', () => {
        expect(warningsForAccountOutflow(undefined, 500)).toEqual([])
        expect(warningsForAccountOutflow(account, 0)).toEqual([])
        expect(warningsForAccountOutflow(account, -500)).toEqual([])
        expect(warningsForAccountOutflow(account, NaN)).toEqual([])
    })
})

describe('warningsForConfirmedDuplicate', () => {
    const spend = { amount: -500, account }

    it('only counts confirmed transactions', () => {
        expect(warningsForConfirmedDuplicate({ status: 'confirmed', payments: [spend] })).toHaveLength(1)
        expect(warningsForConfirmedDuplicate({ status: 'pending', payments: [spend] })).toEqual([])
        expect(warningsForConfirmedDuplicate({ status: 'skipped', payments: [spend] })).toEqual([])
    })

    it('ignores payments that bring money in', () => {
        expect(warningsForConfirmedDuplicate({ status: 'confirmed', payments: [{ amount: 500, account }] })).toEqual([])
    })

    it('warns once for each account that goes negative', () => {
        const other = { ...account, name: 'Other', currentBalance: 10_000 }
        const warnings = warningsForConfirmedDuplicate({
            status: 'confirmed',
            payments: [spend, { amount: -500, account: other }],
        })
        expect(warnings.map((w) => w.accountName)).toEqual([account.name])
    })
})
