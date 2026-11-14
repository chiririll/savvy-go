import { describe, expect, it } from 'vitest'
import { isChangePayment, orderedLines } from './transaction-view'

describe('isChangePayment', () => {
    it('is a positive payment of an expense', () => {
        expect(isChangePayment({ type: 'expense' }, { amount: 2.6 })).toBe(true)
        expect(isChangePayment({ type: 'expense' }, { amount: -52.4 })).toBe(false)
    })

    it('is a negative payment of an income', () => {
        expect(isChangePayment({ type: 'income' }, { amount: -1 })).toBe(true)
        expect(isChangePayment({ type: 'income' }, { amount: 100 })).toBe(false)
    })

    it('never marks the sides of a transfer', () => {
        expect(isChangePayment({ type: 'transfer' }, { amount: 10 })).toBe(false)
        expect(isChangePayment({ type: 'transfer' }, { amount: -10 })).toBe(false)
    })
})

describe('orderedLines', () => {
    it('sorts by seq without changing the input', () => {
        const lines = [{ seq: 2 }, { seq: 1 }]
        expect(orderedLines(lines)).toEqual([{ seq: 1 }, { seq: 2 }])
        expect(lines[0].seq).toBe(2)
    })
})
