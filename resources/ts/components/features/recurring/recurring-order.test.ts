import { describe, expect, it } from 'vitest'

import { recurring } from '@/test/fixtures'
import type { RecurringFrequency } from '@/types'

import { compareRecurring, recurringGroup } from './RecurringRow'

const every = (frequency: RecurringFrequency, interval = 1) => ({
    schedule: { frequency, interval, start: '2026-01-01' },
})

describe('recurring list order', () => {
    it('puts shorter schedules first', () => {
        const frequencies = ['yearly', 'daily', 'monthly', 'weekly'] as const
        const sorted = frequencies.map((frequency) => recurring(every(frequency))).sort(compareRecurring)
        expect(sorted.map((item) => item.schedule.frequency)).toEqual(['daily', 'weekly', 'monthly', 'yearly'])
    })

    it('puts a smaller interval first within a schedule', () => {
        const sorted = [recurring({ id: 1, ...every('monthly', 3) }), recurring({ id: 2, ...every('monthly', 1) })].sort(compareRecurring)
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

    it('puts a rule with no next run after the ones that have one', () => {
        const items = [recurring({ id: 1, nextRunDate: null }), recurring({ id: 2, nextRunDate: '2099-03-01' })]
        expect(items.sort(compareRecurring).map((item) => item.id)).toEqual([2, 1])
    })

    it('groups equal schedules together and tells different intervals apart', () => {
        expect(recurringGroup(recurring(every('monthly', 1))).key).toBe('monthly:1')
        expect(recurringGroup(recurring(every('monthly', 2))).key).toBe('monthly:2')
    })
})
