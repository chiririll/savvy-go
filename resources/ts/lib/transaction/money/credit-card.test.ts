import { describe, expect, it } from 'vitest'

import type { CreditCardInfo } from '@/types'
import { cardProgress, cardTone } from './credit-card'

const info = (overrides: Partial<CreditCardInfo> = {}): CreditCardInfo => ({
    limit: 2000, cycle: 'fixed_day', statementDay: 28, graceDays: 21, used: 500, available: 1500,
    periodStart: '2026-11-01', periodEnd: '2026-11-30', spent: 100,
    statement: { date: '2026-10-28', balance: 400, dueDate: '2026-11-18', paid: 0, amountDue: 400 },
    status: { level: 'ok', amount: 0 },
    ...overrides,
})

describe('cardProgress', () => {
    it('is the share of the limit that is owed, in percent', () => {
        expect(cardProgress(info())).toBe(25)
    })

    it('stops at 100 when the card is over its limit', () => {
        expect(cardProgress(info({ used: 2500, available: -500 }))).toBe(100)
    })

    it('is zero for a credit balance and for no limit', () => {
        expect(cardProgress(info({ used: -30 }))).toBe(0)
        expect(cardProgress(info({ limit: 0 }))).toBe(0)
    })
})

describe('cardTone', () => {
    it('follows the status level', () => {
        expect(cardTone(info())).toBe('ok')
        expect(cardTone(info({ status: { level: 'warning', reason: 'payment_due', amount: 400, date: '2026-11-18' } }))).toBe('warning')
        expect(cardTone(info({ status: { level: 'danger', reason: 'overdue', amount: 400, date: '2026-11-18' } }))).toBe('danger')
    })
})
