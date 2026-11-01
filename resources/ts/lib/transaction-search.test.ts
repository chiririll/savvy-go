import { describe, expect, it } from 'vitest'

import {
    parseTransactionSearch,
    serializeTransactionSearch,
    suggestSearchCompletions,
    type SearchRefs,
} from './transaction-search'

const refs: SearchRefs = {
    accounts: [
        { id: 1, names: ['Cash USD'] },
        { id: 2, names: ['Card'] },
    ],
    categories: [
        { id: 10, names: ['Food', 'Еда'] },
        { id: 11, names: ['Rent'] },
    ],
    tags: [
        { id: 20, names: ['food'] },
        { id: 21, names: ['coffee', 'big coffee'] },
    ],
    currencies: ['USD', 'JPY'],
}

const empty = { q: '', accountIds: [], currency: [], categoryIds: [], tagIds: [], type: null }

describe('parseTransactionSearch', () => {
    it('keeps plain text as the free-text query', () => {
        expect(parseTransactionSearch('  milk   bread ', refs)).toEqual({ ...empty, q: 'milk bread' })
    })

    it('resolves keywords to ids, ignoring case', () => {
        expect(parseTransactionSearch('milk account:card currency:usd type:Expense', refs)).toEqual({
            ...empty,
            q: 'milk',
            accountIds: [2],
            currency: ['USD'],
            type: 'expense',
        })
    })

    it('reads comma-separated values and repeated keys as alternatives', () => {
        const parsed = parseTransactionSearch('tags:food,coffee tags:food category:rent,Еда', refs)
        expect(parsed.tagIds).toEqual([20, 21])
        expect(parsed.categoryIds).toEqual([11, 10])
    })

    it('reads quoted values with spaces', () => {
        expect(parseTransactionSearch('account:"Cash USD" tags:food,"big coffee"', refs)).toEqual({
            ...empty,
            accountIds: [1],
            tagIds: [20, 21],
        })
    })

    it('leaves an unresolved name in the query', () => {
        expect(parseTransactionSearch('account:nope tags:food,nope', refs)).toEqual({
            ...empty,
            q: 'account:nope tags:food,nope',
        })
    })

    it('leaves an unknown key in the query', () => {
        expect(parseTransactionSearch('foo:bar 12:30', refs)).toEqual({ ...empty, q: 'foo:bar 12:30' })
    })

    it('drops the quotes of a phrase without a key', () => {
        expect(parseTransactionSearch('"big coffee" tags:food', refs)).toEqual({
            ...empty,
            q: 'big coffee',
            tagIds: [20],
        })
    })

    it('treats an unterminated quote as the rest of the token', () => {
        expect(parseTransactionSearch('account:"Cash USD', refs).accountIds).toEqual([1])
    })
})

describe('serializeTransactionSearch', () => {
    it('writes the filters back as keywords that parse to the same filters', () => {
        const parsed = {
            q: 'milk',
            accountIds: [1],
            currency: ['USD', 'JPY'],
            categoryIds: [10],
            tagIds: [20, 21],
            type: 'income' as const,
        }
        const text = serializeTransactionSearch(parsed, refs)
        expect(text).toBe('milk type:income account:"Cash USD" currency:USD,JPY category:Food tags:food,coffee')
        expect(parseTransactionSearch(text, refs)).toEqual(parsed)
    })

    it('is empty for no filters', () => {
        expect(serializeTransactionSearch(empty, refs)).toBe('')
    })
})

describe('suggestSearchCompletions', () => {
    it('suggests keywords for a partial key', () => {
        expect(suggestSearchCompletions('milk ta', refs)).toEqual([{ label: 'tags:', insert: 'milk tags:' }])
    })

    it('suggests values for a keyword, skipping the ones already typed', () => {
        expect(suggestSearchCompletions('tags:food,c', refs)).toEqual([
            { label: 'coffee', insert: 'tags:food,coffee ' },
        ])
    })

    it('lists every value right after the colon', () => {
        expect(suggestSearchCompletions('currency:', refs).map((item) => item.label)).toEqual(['USD', 'JPY'])
    })

    it('quotes a value that has spaces', () => {
        expect(suggestSearchCompletions('account:cash', refs)).toEqual([
            { label: 'Cash USD', insert: 'account:"Cash USD" ' },
        ])
    })

    it('suggests nothing for plain words or after a space', () => {
        expect(suggestSearchCompletions('milk', refs)).toEqual([])
        expect(suggestSearchCompletions('tags:food ', refs)).toEqual([])
    })
})
