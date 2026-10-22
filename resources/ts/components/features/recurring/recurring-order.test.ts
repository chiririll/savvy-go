import { describe, expect, it } from 'vitest'

import { recurring } from '@/test/fixtures'

import { compareRecurring, recurringGroup } from './RecurringRow'

describe('recurring list order', () => {
    it('puts shorter schedules first', () => {
        const frequencies = ['yearly', 'daily', 'monthly', 'weekly'] as const
        const sorted = frequencies.map((frequency) => recurring({ frequency })).sort(compareRecurring)
        expect(sorted.map((item) => item.frequency)).toEqual(['daily', 'weekly', 'monthly', 'yearly'])
    })

    it('puts a smaller interval first within a schedule', () => {
        const sorted = [recurring({ id: 1, interval: 3 }), recurring({ id: 2, interval: 1 })].sort(compareRecurring)
        expect(sorted.map((item) => item.id)).toEqual([2, 1])
    })

    it('puts active before paused, then the nearest run first', () => {
        const items = [
            recurring({ id: 1, isActive: false, nextRunDate: '2099-01-01' }),
            recurring({ id: 2, nextRunDate: '2099-03-01' }),
            recurring({ id: 3, nextRunDate: '2099-02-01' }),
        ]
        expect(items.sort(compareRecurring).map((item) => item.id)).toEqual([3, 2, 1])
    })

    it('groups equal schedules together and tells different intervals apart', () => {
        expect(recurringGroup(recurring({ interval: 1 })).key).toBe('monthly:1')
        expect(recurringGroup(recurring({ interval: 2 })).key).toBe('monthly:2')
    })
})
