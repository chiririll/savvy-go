import { describe, expect, it } from 'vitest'
import type { SpaceAction } from '@/types/spaces'
import { groupActions } from './actions'

const act = (id: number, extra: Partial<SpaceAction> = {}): SpaceAction => ({
    id,
    at: '2026-01-01T00:00:00Z',
    source: 'system',
    userId: null,
    entityType: 'transaction',
    entityId: id,
    op: 'create',
    ...extra,
})

describe('groupActions', () => {
    it('folds neighbours of one batch into a group', () => {
        const groups = groupActions([act(4, { batchId: 'a' }), act(3, { batchId: 'a' }), act(2), act(1, { batchId: 'a' })])
        expect(groups.map((g) => g.items.map((i) => i.id))).toEqual([[4, 3], [2], [1]])
    })

    it('keeps different operations of one batch apart', () => {
        const groups = groupActions([act(2, { batchId: 'a', op: 'delete' }), act(1, { batchId: 'a' })])
        expect(groups).toHaveLength(2)
    })
})
