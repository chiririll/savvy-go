export type ItemField = 'name' | 'quantity' | 'price_per_unit'

export type ItemKeyAction =
    | { type: 'addItem' }
    | { type: 'expandToPrice' }
    | { type: 'focusOwn'; field: ItemField }
    | { type: 'removeRow' }
    | { type: 'focusSibling'; delta: 1 | -1 }

interface ItemKeyInput {
    key: string
    shiftKey: boolean
    field: ItemField
    /** Current text of the focused input; only the name field is read. */
    value: string
    index: number
    rowCount: number
}

/** Tab ring inside a row: Shift+Tab from name is left to the browser. */
const FORWARD_TAB: Partial<Record<ItemField, ItemKeyAction>> = {
    name: { type: 'expandToPrice' },
    price_per_unit: { type: 'focusOwn', field: 'quantity' },
}

const BACKWARD_TAB: Partial<Record<ItemField, ItemKeyAction>> = {
    quantity: { type: 'focusOwn', field: 'price_per_unit' },
    price_per_unit: { type: 'focusOwn', field: 'name' },
}

/** What a key press in an item input should do, or null to leave it to the browser. */
export function itemKeyAction({ key, shiftKey, field, value, index, rowCount }: ItemKeyInput): ItemKeyAction | null {
    switch (key) {
        case 'Enter':
            return { type: 'addItem' }
        case 'Tab':
            return (shiftKey ? BACKWARD_TAB : FORWARD_TAB)[field] ?? null
        case 'Backspace':
            return field === 'name' && value === '' && rowCount > 1 ? { type: 'removeRow' } : null
        case 'ArrowDown':
            return index < rowCount - 1 ? { type: 'focusSibling', delta: 1 } : null
        case 'ArrowUp':
            return index > 0 ? { type: 'focusSibling', delta: -1 } : null
        default:
            return null
    }
}
