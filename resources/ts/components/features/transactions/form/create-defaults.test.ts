import { describe, expect, it } from 'vitest'
import { fromCreateDefaults } from './create-defaults'

describe('fromCreateDefaults', () => {
    it('is nothing when the opener asks for nothing', () => {
        expect(fromCreateDefaults(undefined)).toBeUndefined()
    })

    it('turns an account and an amount into the first payment', () => {
        expect(fromCreateDefaults({ type: 'income', account_id: 3, amount: 12.5, description: 'Gift' })).toEqual({
            type: 'income',
            description: 'Gift',
            payments: [{ account_id: 3, amount: 12.5 }],
        })
    })

    it('takes the category as the category of the transaction', () => {
        expect(fromCreateDefaults({ category_id: 2 })).toEqual({ default_category_id: 2 })
    })

    it('leaves the amount empty when only the account is known', () => {
        const defaults = fromCreateDefaults({ account_id: 3 })

        expect(defaults?.payments).toEqual([{ account_id: 3, amount: '' }])
    })

    it('leaves the payment to the form when no account is given', () => {
        expect(fromCreateDefaults({ type: 'expense' })).toEqual({ type: 'expense' })
        expect(fromCreateDefaults({ amount: 5 })?.payments).toEqual([{ account_id: 0, amount: 5 }])
    })

    it('passes a transfer through; the form opens an expense until transfers are back', () => {
        expect(fromCreateDefaults({ type: 'transfer', account_id: 1 })?.type).toBe('transfer')
    })
})
