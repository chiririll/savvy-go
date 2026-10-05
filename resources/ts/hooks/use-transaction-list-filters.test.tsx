// @vitest-environment jsdom
import type { ReactNode } from 'react'
import { act, renderHook } from '@testing-library/react'
import { NuqsTestingAdapter } from 'nuqs/adapters/testing'
import { describe, expect, it } from 'vitest'

import { useTransactionListFilters } from './use-transaction-list-filters'

/** Renders the hook against an in-memory URL that updates like the real one. */
function setup(search = '') {
    const wrapper = ({ children }: { children: ReactNode }) => (
        <NuqsTestingAdapter searchParams={search} hasMemory>
            {children}
        </NuqsTestingAdapter>
    )
    return renderHook(() => useTransactionListFilters(), { wrapper })
}

describe('defaults', () => {
    it('shows confirmed transactions, newest first, from page one', () => {
        const { result } = setup()
        expect(result.current.filters).toEqual({
            per_page: 20,
            page: 1,
            type: undefined,
            sort_by: 'date',
            sort_direction: 'desc',
            category_ids: undefined,
            tag_ids: undefined,
            start_date: undefined,
            end_date: undefined,
            status: 'confirmed',
        })
        expect(result.current.activeFiltersCount).toBe(0)
    })
})

describe('reading the URL', () => {
    it('turns search params into API filters', () => {
        const { result } = setup(
            '?type=income&page=3&categoryIds=1,2&tagIds=7&startDate=2026-01-01&endDate=2026-01-31&sortBy=amount&sortDir=asc',
        )
        expect(result.current.filters).toMatchObject({
            page: 3,
            type: 'income',
            sort_by: 'amount',
            sort_direction: 'asc',
            category_ids: [1, 2],
            tag_ids: [7],
            start_date: '2026-01-01',
            end_date: '2026-01-31',
            status: 'confirmed',
        })
    })

    it('counts each active filter group once', () => {
        expect(setup('?categoryIds=1,2,3').result.current.activeFiltersCount).toBe(1)
        expect(setup('?categoryIds=1&tagIds=2&startDate=2026-01-01&endDate=2026-02-01').result.current.activeFiltersCount).toBe(4)
        // The type and status tabs are not counted as filters.
        expect(setup('?type=expense&status=pending').result.current.activeFiltersCount).toBe(0)
    })

    it('ignores values it does not understand', () => {
        const { result } = setup('?type=bogus&sortBy=colour&status=confirmed')
        expect(result.current.filters).toMatchObject({ type: undefined, sort_by: 'date', status: 'confirmed' })
    })
})

describe('pending transactions', () => {
    it('are listed soonest first by default', () => {
        const { result } = setup('?status=pending')
        expect(result.current.filters).toMatchObject({ status: 'pending', sort_direction: 'asc' })
    })

    it('are still newest first when sorted by amount', () => {
        const { result } = setup('?status=pending&sortBy=amount')
        expect(result.current.filters.sort_direction).toBe('desc')
    })

    it('respect an explicit direction', () => {
        const { result } = setup('?status=pending&sortDir=desc')
        expect(result.current.filters.sort_direction).toBe('desc')
    })
})

describe('changing filters', () => {
    it('setPage only moves the page', () => {
        const { result } = setup('?type=income')
        act(() => { result.current.setPage(4) })
        expect(result.current.filters).toMatchObject({ page: 4, type: 'income' })
    })

    it('goes back to page one when the type changes', () => {
        const { result } = setup('?page=5')
        act(() => { result.current.setType('expense') })
        expect(result.current.filters).toMatchObject({ page: 1, type: 'expense' })
        act(() => { result.current.setType(null) })
        expect(result.current.filters.type).toBeUndefined()
    })

    it('toggles categories and drops the param when none are left', () => {
        const { result } = setup('?page=2')
        act(() => { result.current.toggleCategory(3) })
        act(() => { result.current.toggleCategory(5) })
        expect(result.current.filters).toMatchObject({ category_ids: [3, 5], page: 1 })
        act(() => { result.current.toggleCategory(3) })
        act(() => { result.current.toggleCategory(5) })
        expect(result.current.filters.category_ids).toBeUndefined()
    })

    it('toggles tags', () => {
        const { result } = setup('?tagIds=1,2')
        act(() => { result.current.toggleTag(2) })
        expect(result.current.filters.tag_ids).toEqual([1])
    })

    it('sets and clears a date, treating an empty value as cleared', () => {
        const { result } = setup()
        act(() => { result.current.setDateRange('startDate', '2026-03-01') })
        expect(result.current.filters.start_date).toBe('2026-03-01')
        act(() => { result.current.setDateRange('startDate', '') })
        expect(result.current.filters.start_date).toBeUndefined()
    })

    it('sets the sort and returns to page one', () => {
        const { result } = setup('?page=3')
        act(() => { result.current.setSort('amount', 'asc') })
        expect(result.current.filters).toMatchObject({ sort_by: 'amount', sort_direction: 'asc', page: 1 })
    })

    it('clearFilters removes filters but keeps type, sort and status', () => {
        const { result } = setup('?type=income&sortBy=amount&sortDir=asc&categoryIds=1&tagIds=2&startDate=2026-01-01&endDate=2026-01-31&page=4')
        act(() => { result.current.clearFilters() })
        expect(result.current.filters).toMatchObject({
            type: 'income',
            sort_by: 'amount',
            sort_direction: 'asc',
            category_ids: undefined,
            tag_ids: undefined,
            start_date: undefined,
            end_date: undefined,
            page: 1,
        })
        expect(result.current.activeFiltersCount).toBe(0)
    })
})

describe('switching to pending', () => {
    it('resets a date sort direction so pending starts soonest first', () => {
        const { result } = setup('?sortDir=desc&page=2')
        act(() => { result.current.setStatus('pending') })
        expect(result.current.filters).toMatchObject({ status: 'pending', sort_direction: 'asc', page: 1 })
    })

    it('keeps the direction chosen for an amount sort', () => {
        const { result } = setup('?sortBy=amount&sortDir=asc')
        act(() => { result.current.setStatus('pending') })
        expect(result.current.filters).toMatchObject({ status: 'pending', sort_by: 'amount', sort_direction: 'asc' })
    })

    it('goes back to confirmed with null', () => {
        const { result } = setup('?status=pending')
        act(() => { result.current.setStatus(null) })
        expect(result.current.filters.status).toBe('confirmed')
    })
})
