import { describe, expect, it } from 'vitest'

import {
    collectNegativeBalanceWarnings,
    warningIfResultNegative,
    warningsForAccountOutflow,
    warningsForConfirmedDuplicate,
    warningsForTransactionOutflow,
} from './negative-balance'

const account = { name: 'Cash', currentBalance: 100 }

describe('warningIfResultNegative', () => {
    it('warns when a spend takes the balance below zero', () => {
        expect(warningIfResultNegative({ accountName: 'Cash', currentBalance: 100, resultingBalance: -20, currency: 'USD' }))
            .toEqual({ accountName: 'Cash', currentBalance: 100, resultingBalance: -20, currency: 'USD' })
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
        expect(warningIfResultNegative({ accountName: 'Cash', currentBalance: -50, resultingBalance: -60 })).not.toBeNull()
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

describe('warningsForTransactionOutflow', () => {
    it.each(['expense', 'transfer', 'debt_payment', 'debt_lend'])('checks %s', (type) => {
        expect(warningsForTransactionOutflow({ type, amount: 500, account })).toHaveLength(1)
    })

    it.each(['income', 'debt_collection', 'debt_borrow', 'transfer_in'])('ignores %s', (type) => {
        expect(warningsForTransactionOutflow({ type, amount: 500, account })).toEqual([])
    })
})

describe('warningsForConfirmedDuplicate', () => {
    it('only counts confirmed transactions', () => {
        const tx = { type: 'expense', amount: 500, account }
        expect(warningsForConfirmedDuplicate({ ...tx, status: 'confirmed' })).toHaveLength(1)
        expect(warningsForConfirmedDuplicate({ ...tx, status: 'pending' })).toEqual([])
        expect(warningsForConfirmedDuplicate({ ...tx, status: 'skipped' })).toEqual([])
    })
})
