import { describe, expect, it } from 'vitest'

import { swapUpdates } from './transfer-swap'

describe('swapUpdates', () => {
    it('does nothing when no account is set', () => {
        expect(swapUpdates({ account_id: 0, to_account_id: '' })).toBeNull()
    })

    it('moves a lone account to the other side without touching the amounts', () => {
        expect(swapUpdates({ account_id: 3, amount: 10, to_amount: 12 })).toEqual({
            to_account_id: 3,
            account_id: null,
        })
        expect(swapUpdates({ to_account_id: '4', amount: 10 })).toEqual({ to_account_id: null, account_id: 4 })
    })

    it('swaps the amounts and inverts the rate when both accounts are set', () => {
        expect(swapUpdates({ account_id: 1, to_account_id: 2, amount: 10, to_amount: 25, exchange_rate: 2.5 })).toEqual(
            { to_account_id: 1, account_id: 2, amount: 25, to_amount: 10, exchange_rate: 0.4 },
        )
    })

    it('keeps the amounts when nothing was received yet', () => {
        const updates = swapUpdates({ account_id: 1, to_account_id: 2, amount: 10, to_amount: '' })
        expect(updates).toEqual({ to_account_id: 1, account_id: 2 })
    })

    it('leaves an unusable rate alone', () => {
        expect(swapUpdates({ account_id: 1, to_account_id: 2, exchange_rate: 0 })).toEqual({
            to_account_id: 1,
            account_id: 2,
        })
    })
})
