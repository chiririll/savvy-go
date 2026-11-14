import { beforeEach, describe, expect, it } from 'vitest'

import { useCurrenciesStore } from '@/stores/currencies'
import { accounts, categories, currencies, line, payment, transaction } from '@/test/fixtures'
import { lineCurrency, primaryCategory } from './transaction-money'

const [usd, eur, jpy] = currencies
const [cash, euroCard, yenWallet] = accounts

beforeEach(() => useCurrenciesStore.getState().setAll(currencies))

describe('lineCurrency', () => {
    it('is the currency of a payment account that has it', () => {
        const tx = transaction({ payments: [payment({ account: euroCard, accountId: 2, amount: -5 })] })
        expect(lineCurrency(tx, line({ currencyId: eur.id }))).toBe(euroCard.currency)
    })

    it('finds the currency among the space currencies when no payment account has it', () => {
        const tx = transaction({ lines: [line({ currencyId: jpy.id, amount: 300 })] })
        expect(lineCurrency(tx, tx.lines[0])).toBe(jpy)
    })
})

describe('primaryCategory', () => {
    const [groceries, salary, transport] = categories

    it('is the category of the largest line by base value', () => {
        const tx = transaction({
            lines: [
                line({ category: groceries, amount: 5 }),
                line({ seq: 1, category: transport, categoryId: transport.id, amount: 20 }),
            ],
        })
        expect(primaryCategory(tx)).toBe(transport)
    })

    it('weighs lines in other currencies by rate', () => {
        const tx = transaction({
            payments: [payment(), payment({ account: yenWallet, accountId: 3 })],
            lines: [
                line({ category: groceries, amount: 100 }), // 100 USD
                line({ seq: 1, category: salary, categoryId: salary.id, currencyId: jpy.id, amount: 1000 }), // 5 USD
            ],
        })
        expect(primaryCategory(tx)).toBe(groceries)
    })

    it('takes the first line on a tie and skips uncategorised lines', () => {
        const tied = transaction({
            lines: [
                line({ category: null, categoryId: null, amount: 99 }),
                line({ seq: 1, category: groceries, amount: 10 }),
                line({ seq: 2, category: transport, categoryId: transport.id, amount: 10 }),
            ],
        })
        expect(primaryCategory(tied)).toBe(groceries)
        expect(primaryCategory(transaction({ lines: [] }))).toBeNull()
    })
})
