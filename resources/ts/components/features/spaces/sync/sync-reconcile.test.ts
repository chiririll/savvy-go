import { describe, expect, it } from 'vitest'

import type { SyncCategoryConflict, SyncDiff } from '@/types/spaces'

import {
    buildDecisions,
    buildDefaultDecisions,
    categoryKey,
    emptyChoices,
    isComplete,
    isIdentical,
    needsSuccessor,
    successorNames,
    summarize,
    syncSteps,
    tagKey,
} from './sync-reconcile'

const category = (extra: Partial<SyncCategoryConflict> = {}): SyncCategoryConflict =>
    ({
        type: 'expense',
        name: 'Food',
        present: [
            { spaceId: 1, icon: 'a', color: 'red', transactions: 3 },
            { spaceId: 2, icon: 'b', color: 'red', transactions: 0 },
        ],
        missing: [3],
        ...extra,
    }) as SyncCategoryConflict

const diff = (extra: Partial<SyncDiff> = {}): SyncDiff =>
    ({
        spaces: [],
        categories: [category()],
        tags: [{ name: 'Trip', present: [], missing: [] }],
        defaults: [],
        ...extra,
    }) as unknown as SyncDiff

describe('sync reconcile decisions', () => {
    it('keeps everything and takes the first look until told otherwise', () => {
        const [cat, tag] = buildDecisions(diff(), emptyChoices())
        expect(cat).toMatchObject({ kind: 'categories', keep: true, from_space_id: 1, successor: undefined })
        expect(tag).toMatchObject({ kind: 'tags', name: 'Trip', keep: true })
    })

    it('sends the successor only for a dropped category', () => {
        const choices = emptyChoices()
        choices.keep[categoryKey(category())] = false
        choices.successor[categoryKey(category())] = 'Meals'
        const [cat] = buildDecisions(diff(), choices)
        expect(cat).toMatchObject({ keep: false, from_space_id: undefined, successor: 'Meals' })
    })

    it('asks for no look when the spaces agree', () => {
        const same = category({
            present: [
                { spaceId: 1, icon: 'a', color: 'red', transactions: 0 },
                { spaceId: 2, icon: 'a', color: 'red', transactions: 0 },
            ],
        } as Partial<SyncCategoryConflict>)
        const [cat] = buildDecisions(diff({ categories: [same] }), emptyChoices())
        expect(cat).toMatchObject({ from_space_id: undefined })
    })
})

describe('sync reconcile completeness', () => {
    it('blocks Apply until a dropped category with transactions has a successor', () => {
        const choices = emptyChoices()
        choices.keep[categoryKey(category())] = false
        expect(needsSuccessor(category(), choices)).toBe(true)
        expect(isComplete(diff(), choices)).toBe(false)

        choices.successor[categoryKey(category())] = 'Meals'
        expect(isComplete(diff(), choices)).toBe(true)
    })

    it('needs no successor for an empty category', () => {
        const empty = category({ present: [{ spaceId: 1, transactions: 0 }] } as Partial<SyncCategoryConflict>)
        const choices = emptyChoices()
        choices.keep[categoryKey(empty)] = false
        expect(needsSuccessor(empty, choices)).toBe(false)
    })

    it('knows when the spaces are identical', () => {
        expect(isIdentical(undefined)).toBe(false)
        expect(isIdentical(diff({ categories: [], tags: [] }))).toBe(true)
        expect(isIdentical(diff())).toBe(false)
    })
})

describe('sync reconcile defaults and successors', () => {
    it('defaults to the first distinct name unless one is picked', () => {
        const data = diff({
            defaults: [
                {
                    type: 'expense',
                    defaults: [
                        { spaceId: 1, name: 'Food' },
                        { spaceId: 2, name: '' },
                        { spaceId: 3, name: 'Food' },
                    ],
                },
            ],
        } as unknown as Partial<SyncDiff>)
        expect(buildDefaultDecisions(data, emptyChoices())).toEqual([{ type: 'expense', name: 'Food' }])

        const choices = emptyChoices()
        choices.defaults.expense = 'Other'
        expect(buildDefaultDecisions(data, choices)).toEqual([{ type: 'expense', name: 'Other' }])
    })

    it('offers only categories of the type that are not themselves in conflict', () => {
        const mine: Pick<SyncCategoryConflict, 'type' | 'name'>[] = [
            { type: 'expense', name: 'Food' },
            { type: 'expense', name: 'Meals' },
            { type: 'income', name: 'Pay' },
        ]
        expect(successorNames(mine, diff(), 'expense')).toEqual(['Meals'])
    })
})

describe('sync reconcile steps', () => {
    it('shows only the steps that have something to decide, summary last', () => {
        expect(syncSteps(diff(), emptyChoices())).toEqual(['categories', 'tags', 'summary'])
        expect(syncSteps(diff({ tags: [] }), emptyChoices())).toEqual(['categories', 'summary'])
        expect(syncSteps(diff({ categories: [] }), emptyChoices())).toEqual(['tags', 'summary'])
        expect(syncSteps(diff({ categories: [], tags: [] }), emptyChoices())).toEqual(['summary'])
    })

    it('adds the merge step only when a dropped category holds transactions', () => {
        const choices = emptyChoices()
        choices.keep[categoryKey(category())] = false
        expect(syncSteps(diff(), choices)).toEqual(['categories', 'merge', 'tags', 'summary'])

        const empty = category({
            present: [{ spaceId: 1, icon: 'a', color: 'red', transactions: 0 }],
        } as Partial<SyncCategoryConflict>)
        const none = emptyChoices()
        none.keep[categoryKey(empty)] = false
        expect(syncSteps(diff({ categories: [empty] }), none)).not.toContain('merge')
    })

    it('counts what will be kept and deleted', () => {
        const other = category({ name: 'Fun' })
        const data = diff({ categories: [category(), other] })
        const choices = emptyChoices()
        choices.keep[categoryKey(other)] = false
        choices.keep[tagKey({ name: 'Trip' })] = false
        expect(summarize(data, choices)).toEqual({ categoriesKept: 1, categoriesDropped: 1, tagsKept: 0, tagsDropped: 1 })
    })
})
