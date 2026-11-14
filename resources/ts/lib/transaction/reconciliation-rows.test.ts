import { describe, expect, it } from 'vitest'

import type { Account, Currency } from '@/types'
import {
    addRow,
    applyComputed,
    canSave,
    editRow,
    initialRows,
    removeRow,
    removedAccounts,
    rowDifference,
    toPayload,
} from './reconciliation-rows'

const eur = { id: 1, code: 'EUR', decimals: 2 } as Currency
const jpy = { id: 2, code: 'JPY', decimals: 0 } as Currency

function account(id: number, type: Account['type'], currentBalance: number, currency: Currency | null = eur): Account {
    return { id, name: `A${id}`, type, currentBalance, currency } as Account
}

const accounts = [
    account(1, 'bank', 10.5),
    account(2, 'debt', 5),
    account(3, 'cash', 1000, jpy),
    account(4, 'space', 1),
    account(5, 'credit', -0.1),
]

describe('reconcile rows', () => {
    it('starts with every reconcilable account, in list order, at its current balance', () => {
        expect(initialRows(accounts)).toEqual([
            { accountId: 1, amount: '10.50', computed: '10.50', edited: false },
            { accountId: 3, amount: '1000', computed: '1000', edited: false },
            { accountId: 5, amount: '-0.10', computed: '-0.10', edited: false },
        ])
    })

    it('removes a row and offers it back, placing it where the list has it', () => {
        const rows = removeRow(initialRows(accounts), 3)
        expect(removedAccounts(accounts, rows).map((a) => a.id)).toEqual([3])
        const back = addRow(rows, accounts, 3, '999')
        expect(back.map((r) => r.accountId)).toEqual([1, 3, 5])
        expect(back[1]).toEqual({ accountId: 3, amount: '999', computed: '999', edited: false })
        expect(removedAccounts(accounts, back)).toEqual([])
    })

    it('does not add a row twice', () => {
        const rows = initialRows(accounts)
        expect(addRow(rows, accounts, 1, '1.00')).toEqual(rows)
    })

    it('refreshes rows that were not edited and keeps edited ones', () => {
        let rows = editRow(initialRows(accounts), 1, '12.34')
        rows = applyComputed(rows, new Map([[1, '20.00'], [3, '1500'], [5, '0.00']]))
        expect(rows[0]).toEqual({ accountId: 1, amount: '12.34', computed: '20.00', edited: true })
        expect(rows[1]).toEqual({ accountId: 3, amount: '1500', computed: '1500', edited: false })
        expect(rows[2].amount).toBe('0.00')
    })

    it('leaves rows without a new balance as they are', () => {
        const rows = initialRows(accounts)
        expect(applyComputed(rows, new Map())).toEqual(rows)
    })

    it('computes the difference exactly', () => {
        const row = { accountId: 1, amount: '0.30', computed: '0.10', edited: true }
        expect(rowDifference(row, 2)).toBe('0.20')
        expect(rowDifference({ ...row, amount: '0.1', computed: '0.30' }, 2)).toBe('-0.20')
        expect(rowDifference({ ...row, amount: '' }, 2)).toBeNull()
        expect(rowDifference({ ...row, computed: null }, 2)).toBeNull()
    })

    it('can be saved with a date not in the future, a row and complete amounts', () => {
        const rows = initialRows(accounts)
        const decimals = new Map([[1, 2], [3, 0], [5, 2]])
        expect(canSave('2026-11-08', '2026-11-08', rows, decimals)).toBe(true)
        expect(canSave('2026-11-09', '2026-11-08', rows, decimals)).toBe(false)
        expect(canSave('', '2026-11-08', rows, decimals)).toBe(false)
        expect(canSave('2026-11-08', '2026-11-08', [], decimals)).toBe(false)
        expect(canSave('2026-11-08', '2026-11-08', editRow(rows, 1, '.'), decimals)).toBe(false)
        expect(canSave('2026-11-08', '2026-11-08', editRow(rows, 1, ''), decimals)).toBe(false)
    })

    it('maps the rows to the batch', () => {
        const rows = editRow(initialRows(accounts), 3, '1,5'.replace(',', '.'))
        expect(toPayload('2026-11-08', rows)).toEqual({
            date: '2026-11-08',
            items: [
                { accountId: 1, actualBalance: 10.5 },
                { accountId: 3, actualBalance: 1.5 },
                { accountId: 5, actualBalance: -0.1 },
            ],
        })
    })
})
