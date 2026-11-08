import { describe, expect, it } from 'vitest'
import { toRecurringDefaults } from './recurring-defaults'

describe('toRecurringDefaults', () => {
    it('starts with a monthly expense with no date, active, today as the start', () => {
        const values = toRecurringDefaults()

        expect(values).toMatchObject({
            type: 'expense',
            date: null,
            estimated: false,
            is_active: true,
            frequency: 'monthly',
            interval: 1,
            end_date: null,
            payments: [{ account_id: 0 }],
        })
        expect(values.start_date).toMatch(/^\d{4}-\d{2}-\d{2}$/)
    })

    it('takes the day of the week and of the month from today', () => {
        const values = toRecurringDefaults()
        const today = new Date()

        expect(values.day_of_week).toBe(today.getDay())
        expect(values.day_of_month).toBe(today.getDate())
    })

    it('keeps what the opener asks for, never a date for the template', () => {
        const values = toRecurringDefaults({ type: 'income', date: '2026-01-01', frequency: 'weekly', is_active: false })

        expect(values).toMatchObject({ type: 'income', date: null, frequency: 'weekly', is_active: false })
    })
})
