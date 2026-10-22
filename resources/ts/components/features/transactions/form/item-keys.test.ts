import { describe, expect, it } from 'vitest'

import { itemKeyAction, type ItemField } from './item-keys'

const press = (
    key: string,
    field: ItemField,
    extra: { shiftKey?: boolean; value?: string; index?: number; rowCount?: number } = {},
) => itemKeyAction({ key, field, shiftKey: false, value: 'x', index: 1, rowCount: 3, ...extra })

describe('itemKeyAction', () => {
    it('adds an item on Enter in any field', () => {
        expect(press('Enter', 'name')).toEqual({ type: 'addItem' })
        expect(press('Enter', 'quantity')).toEqual({ type: 'addItem' })
    })

    it('walks name -> price -> quantity on Tab', () => {
        expect(press('Tab', 'name')).toEqual({ type: 'expandToPrice' })
        expect(press('Tab', 'price_per_unit')).toEqual({ type: 'focusOwn', field: 'quantity' })
    })

    it('walks quantity -> price -> name on Shift+Tab', () => {
        expect(press('Tab', 'quantity', { shiftKey: true })).toEqual({ type: 'focusOwn', field: 'price_per_unit' })
        expect(press('Tab', 'price_per_unit', { shiftKey: true })).toEqual({ type: 'focusOwn', field: 'name' })
    })

    it('leaves the Tab edges to the browser', () => {
        expect(press('Tab', 'quantity')).toBeNull()
        expect(press('Tab', 'name', { shiftKey: true })).toBeNull()
    })

    it('removes a row on Backspace in an empty name, except the last', () => {
        expect(press('Backspace', 'name', { value: '' })).toEqual({ type: 'removeRow' })
        expect(press('Backspace', 'name', { value: 'a' })).toBeNull()
        expect(press('Backspace', 'name', { value: '', rowCount: 1 })).toBeNull()
        expect(press('Backspace', 'quantity', { value: '' })).toBeNull()
    })

    it('moves between rows with the arrows, stopping at the ends', () => {
        expect(press('ArrowDown', 'name')).toEqual({ type: 'focusSibling', delta: 1 })
        expect(press('ArrowUp', 'name')).toEqual({ type: 'focusSibling', delta: -1 })
        expect(press('ArrowDown', 'name', { index: 2 })).toBeNull()
        expect(press('ArrowUp', 'name', { index: 0 })).toBeNull()
    })

    it('ignores other keys', () => {
        expect(press('a', 'name')).toBeNull()
    })
})
