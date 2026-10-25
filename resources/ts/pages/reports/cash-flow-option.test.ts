import { describe, expect, it } from 'vitest'

import type { CashFlowDataPoint } from '@/api/reports'

import { cashFlowState } from './cash-flow-option'

const point = (income: number, expenses: number): CashFlowDataPoint =>
    ({ date: '2026-01-01', income, expenses, balance: income - expenses }) as CashFlowDataPoint

describe('cashFlowState', () => {
    it('reports missing data', () => {
        expect(cashFlowState(undefined)).toBe('noData')
        expect(cashFlowState([])).toBe('noData')
    })

    it('names the missing side', () => {
        expect(cashFlowState([point(0, 0)])).toBe('noIncomeOrExpenses')
        expect(cashFlowState([point(0, 5)])).toBe('noIncome')
        expect(cashFlowState([point(5, 0)])).toBe('noExpenses')
    })

    it('is ok when any period has both sides across the range', () => {
        expect(cashFlowState([point(5, 0), point(0, 3)])).toBe('ok')
    })
})
