import { describe, expect, it } from 'vitest'
import type { TFunction } from 'i18next'
import { topTransactionAccount, topTransactionCategory, topTransactionTitle } from './top-transaction'

const base = {
    description: 'Market',
    category: { id: 1, name: 'Food', icon: '🛒', color: '#000' },
    account: { id: 1, name: 'Cash' },
    category_count: 1,
    account_count: 1,
}
const t = ((key: string, o?: { count: number }) => `${key}:${o?.count}`) as unknown as TFunction<'pages'>

describe('top transaction labels', () => {
    it('title falls back to the category, then to the given text', () => {
        expect(topTransactionTitle(base, 'None')).toBe('Market')
        expect(topTransactionTitle({ ...base, description: null }, 'None')).toBe('Food')
        expect(topTransactionTitle({ ...base, description: '', category: null }, 'None')).toBe('None')
    })

    it('category shows the name, the count for several, nothing for none', () => {
        expect(topTransactionCategory(base, t)).toBe('Food')
        expect(topTransactionCategory({ ...base, category_count: 3 }, t)).toBe('transactions.categoriesCount:3')
        expect(topTransactionCategory({ ...base, category: null, category_count: 0 }, t)).toBe('')
    })

    it('account adds +N for the other accounts', () => {
        expect(topTransactionAccount(base)).toBe('Cash')
        expect(topTransactionAccount({ ...base, account_count: 3 })).toBe('Cash +2')
    })
})
