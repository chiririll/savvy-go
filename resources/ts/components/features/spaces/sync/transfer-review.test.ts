import { describe, expect, it } from 'vitest'
import type { SpaceTransfer } from '@/types/spaces'
import { sidesForSpace, transferNeedsDecision } from './transfer-review'

const transfer = (overrides: Partial<SpaceTransfer> = {}): SpaceTransfer => ({
    uuid: 'a',
    from: { amount: '200.00', currency: 'EUR' },
    to: { amount: '23400', currency: 'RSD' },
    outgoing: true,
    date: '2026-11-03',
    description: null,
    version: 1,
    status: 'confirmed',
    review: null,
    needsAttention: false,
    frozen: false,
    verified: true,
    deleted: false,
    ...overrides,
})

describe('transferNeedsDecision', () => {
    it('leaves a confirmed, verified, linked transfer out', () => {
        expect(transferNeedsDecision(transfer())).toBe(false)
    })

    it.each([
        ['in review', { status: 'in_review' as const }],
        ['missing an account', { needsAttention: true }],
        ['unverified', { verified: false }],
        ['frozen', { frozen: true }],
    ])('lists a transfer that is %s', (_, overrides) => {
        expect(transferNeedsDecision(transfer(overrides))).toBe(true)
    })
})

describe('sidesForSpace', () => {
    it('puts this space first, each side in its own currency', () => {
        expect(sidesForSpace(transfer(), true)).toEqual(['−200.00 EUR', '+23400 RSD'])
        expect(sidesForSpace(transfer(), false)).toEqual(['+23400 RSD', '−200.00 EUR'])
    })
})
