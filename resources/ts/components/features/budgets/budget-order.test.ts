import { describe, expect, it } from 'vitest'

import { budget } from '@/test/fixtures'

import { budgetGroup, compareBudgets } from './BudgetRow'

describe('budget list order', () => {
    it('puts shorter periods first', () => {
        const periods = ['yearly', 'weekly', 'one_time', 'monthly'] as const
        const sorted = periods.map((period) => budget({ period })).sort(compareBudgets)
        expect(sorted.map((item) => item.period)).toEqual(['weekly', 'monthly', 'yearly', 'one_time'])
    })

    it('puts active budgets before inactive ones of the same period', () => {
        const sorted = [budget({ id: 1, isActive: false }), budget({ id: 2, isActive: true })].sort(compareBudgets)
        expect(sorted.map((item) => item.id)).toEqual([2, 1])
    })

    it('groups by period', () => {
        expect(budgetGroup(budget({ period: 'yearly' })).key).toBe('yearly')
    })
})
