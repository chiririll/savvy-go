// @vitest-environment jsdom
import { renderHook } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { Category } from '@/types'
import { useTransactionPartyDefaults } from './use-transaction-party-defaults'

const cat = (id: number, over: Partial<Category>) => ({ id, type: 'expense', isDefault: false, transactionsCount: 0, ...over }) as Category

function run(categories: Category[]) {
    const setValue = vi.fn()
    const values: Record<string, unknown> = { type: 'expense', account_id: null, category_id: null }
    renderHook(() => useTransactionPartyDefaults({ watch: (n) => values[n], setValue }, undefined, categories))
    return setValue
}

describe('useTransactionPartyDefaults', () => {
    it('preselects the default category, not the first or most used', () => {
        const setValue = run([cat(1, { transactionsCount: 9 }), cat(2, { isDefault: true }), cat(3, { type: 'income', isDefault: true })])
        expect(setValue).toHaveBeenCalledWith('category_id', 2)
    })

    it('falls back to the most used category when none is default', () => {
        const setValue = run([cat(1, { transactionsCount: 1 }), cat(2, { transactionsCount: 5 })])
        expect(setValue).toHaveBeenCalledWith('category_id', 2)
    })
})
