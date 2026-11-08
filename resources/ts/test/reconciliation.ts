// Sample reconciliation data for stories.
import type { ReconciliationDay, ReconciliationPoint } from '@/types'

export const reconciliationPoint = (overrides: Partial<ReconciliationPoint> = {}): ReconciliationPoint => ({
    id: 1,
    accountId: 1,
    date: '2020-05-04',
    actualBalance: 130,
    expectedBalance: 120.5,
    difference: 9.5,
    note: null,
    differenceBase: 9.5,
    ...overrides,
})

export const reconciliationDay = (overrides: Partial<ReconciliationDay> = {}): ReconciliationDay => ({
    date: '2020-05-04',
    differenceBase: 9.5,
    unconverted: false,
    points: [reconciliationPoint()],
    ...overrides,
})
