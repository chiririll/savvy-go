import { describe, expect, it } from 'vitest'
import { account, currencies, currency } from '@/test/fixtures'
import { followedAmount, lockBlocker, type LockValues } from './payment-lock'

const [usd, eur, jpy, kwd] = currencies
const usdCash = account({ id: 1, currency: usd })
const usdCard = account({ id: 2, currency: usd })
const eurCard = account({ id: 3, currency: eur })
const jpyCash = account({ id: 4, currency: jpy })
const kwdCash = account({ id: 5, currency: kwd })
const btc = currency({ id: 9, code: 'BTC', name: 'Bitcoin', symbol: '₿', decimals: 8, isBase: false, rate: 60000 })
const btcWallet = account({ id: 6, currency: btc })
const accounts = [usdCash, usdCard, eurCard, jpyCash, kwdCash, btcWallet]

const pay = (account_id: number, amount: number | string = 10) => ({ account_id, amount: amount as number })
const line = (amount: number | string, currency_id: number | null = null) => ({ category_id: 1, currency_id, amount: amount as number, note: '' })

describe('followedAmount', () => {
    it('is the sum of the lines, exact', () => {
        const values: LockValues = { payments: [pay(1)], lines: [line('0.1'), line('0.2')] }
        expect(followedAmount(values, accounts, 0)).toBe('0.30')
    })

    it('counts the lines in the currency of the first payment when they name none', () => {
        const values: LockValues = { payments: [pay(1), pay(3)], lines: [line('5'), line('7', 2), line('2', 1)] }
        expect(followedAmount(values, accounts, 0)).toBe('7.00')
        expect(followedAmount(values, accounts, 1)).toBe('7.00')
    })

    it('adds the change in the same currency', () => {
        const values: LockValues = { payments: [pay(1)], change: [pay(2, '3.50')], lines: [line('10')] }
        expect(followedAmount(values, accounts, 0)).toBe('13.50')
    })

    it('cannot follow when change came back in a currency no payment is in', () => {
        const values: LockValues = { payments: [pay(1)], change: [pay(3, '500')], lines: [line('10')] }
        expect(followedAmount(values, accounts, 0)).toBeNull()
        expect(lockBlocker(values, accounts, 0)).toBe('foreignChange')
    })

    it('is not stopped by change that is empty or zero', () => {
        expect(followedAmount({ payments: [pay(1)], change: [pay(3, '')], lines: [line('10')] }, accounts, 0)).toBe('10.00')
        expect(followedAmount({ payments: [pay(1)], change: [pay(3, '0')], lines: [line('10')] }, accounts, 0)).toBe('10.00')
    })

    it('leaves the change in the currency of another payment to that payment', () => {
        const values: LockValues = { payments: [pay(1), pay(3)], change: [pay(3, '5')], lines: [line('10'), line('7', 2)] }
        expect(lockBlocker(values, accounts, 0)).toBeNull()
        expect(followedAmount(values, accounts, 0)).toBe('10.00')
        expect(followedAmount(values, accounts, 1)).toBe('12.00')
    })

    it('uses the decimals of the currency', () => {
        const values: LockValues = { payments: [pay(4)], lines: [line('100'), line('250')] }
        expect(followedAmount(values, accounts, 0)).toBe('350')
    })

    it('keeps three decimals of a dinar and none of a yen', () => {
        expect(followedAmount({ payments: [pay(5)], lines: [line('0.1'), line('0.002')] }, accounts, 0)).toBe('0.102')
        expect(followedAmount({ payments: [pay(5)], change: [], lines: [line('1.5')] }, accounts, 0)).toBe('1.500')
        expect(followedAmount({ payments: [pay(4)], lines: [line('150'), line('1.5')] }, accounts, 0)).toBe('150')
    })

    it('keeps eight decimals of a bitcoin, exact', () => {
        const values: LockValues = { payments: [pay(6)], lines: [line('0.00000001'), line('0.1'), line('0.2')] }
        expect(followedAmount(values, accounts, 0)).toBe('0.30000001')
        expect(followedAmount({ payments: [pay(6)], lines: [line('21000000')] }, accounts, 0)).toBe('21000000.00000000')
    })

    it('counts a line in its own currency for each payment', () => {
        const values: LockValues = { payments: [pay(1), pay(4), pay(5)], lines: [line('1.25'), line('300', 3), line('0.125', 4)] }
        expect(followedAmount(values, accounts, 0)).toBe('1.25')
        expect(followedAmount(values, accounts, 1)).toBe('300')
        expect(followedAmount(values, accounts, 2)).toBe('0.125')
    })

    it('skips empty and broken amounts', () => {
        const values: LockValues = { payments: [pay(1)], lines: [line(''), line('4'), line('1+')] }
        expect(followedAmount(values, accounts, 0)).toBe('4.00')
    })

    it('is empty while the lines in its currency have no amount yet', () => {
        const values: LockValues = { payments: [pay(1)], lines: [line('')] }
        expect(followedAmount(values, accounts, 0)).toBe('')
    })

    it('cannot follow without lines, or without a line in its currency', () => {
        expect(followedAmount({ payments: [pay(1)], lines: [] }, accounts, 0)).toBeNull()
        expect(followedAmount({ payments: [pay(1)] }, accounts, 0)).toBeNull()
        expect(followedAmount({ payments: [pay(1), pay(3)], lines: [line('5')] }, accounts, 1)).toBeNull()
    })

    it('cannot follow when another payment is in the same currency', () => {
        const values: LockValues = { payments: [pay(1), pay(2)], lines: [line('5')] }
        expect(followedAmount(values, accounts, 0)).toBeNull()
        expect(followedAmount(values, accounts, 1)).toBeNull()
    })

    it('cannot follow without an account', () => {
        expect(followedAmount({ payments: [pay(99)], lines: [line('5')] }, accounts, 0)).toBeNull()
    })
})

describe('lockBlocker', () => {
    it('says why a row stays locked', () => {
        expect(lockBlocker({ payments: [pay(1), pay(2)], lines: [line('5')] }, accounts, 0)).toBe('several')
        expect(lockBlocker({ payments: [pay(1), pay(3)], lines: [line('5')] }, accounts, 1)).toBe('noLine')
        expect(lockBlocker({ payments: [pay(1)], lines: [] }, accounts, 0)).toBe('noLine')
    })

    it('is null when the row can follow', () => {
        expect(lockBlocker({ payments: [pay(1), pay(3)], lines: [line('5')] }, accounts, 0)).toBeNull()
    })
})
