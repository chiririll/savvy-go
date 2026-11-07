import { describe, expect, it } from 'vitest'
import type { Category } from '@/types'
import { pickDefaultCategory } from './default-category'

const cat = (id: number, over: Partial<Category>) => ({ id, type: 'expense', isDefault: false, transactionsCount: 0, ...over }) as Category

describe('pickDefaultCategory', () => {
    it('picks the default category, not the first or most used', () => {
        const picked = pickDefaultCategory(
            [cat(1, { transactionsCount: 9 }), cat(2, { isDefault: true }), cat(3, { type: 'income', isDefault: true })],
            'expense',
        )
        expect(picked?.id).toBe(2)
    })

    it('falls back to the most used category when none is default', () => {
        expect(pickDefaultCategory([cat(1, { transactionsCount: 1 }), cat(2, { transactionsCount: 5 })], 'expense')?.id).toBe(2)
    })

    it('has none for a type with no categories, or before they load', () => {
        expect(pickDefaultCategory([cat(1, { type: 'income' })], 'expense')).toBeUndefined()
        expect(pickDefaultCategory(undefined, 'expense')).toBeUndefined()
    })
})
