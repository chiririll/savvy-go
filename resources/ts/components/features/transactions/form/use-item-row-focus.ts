import { useCallback, useEffect, useLayoutEffect, useRef, type KeyboardEvent } from 'react'

import { itemKeyAction, type ItemField } from './item-keys'

interface UseItemRowFocusOptions {
    index: number
    rowCount: number
    expanded: boolean
    /** A new quantity or price error opens the row so it can be seen. */
    quantityError: unknown
    priceError: unknown
    onExpand: () => void
    onToggle: () => void
    onRemove: () => void
    onAddItem: () => void
    onFocusField: (index: number, field: ItemField) => void
}

/** Refs, keyboard moves and fold behaviour of one item row; the price field gets focus once the row opens. */
export function useItemRowFocus({
    index,
    rowCount,
    expanded,
    quantityError,
    priceError,
    onExpand,
    onToggle,
    onRemove,
    onAddItem,
    onFocusField,
}: UseItemRowFocusOptions) {
    const nameRef = useRef<HTMLInputElement | null>(null)
    const quantityRef = useRef<HTMLInputElement | null>(null)
    const priceRef = useRef<HTMLInputElement | null>(null)
    const pendingPriceFocus = useRef(false)
    const ownInputs: Record<ItemField, typeof nameRef> = {
        name: nameRef,
        quantity: quantityRef,
        price_per_unit: priceRef,
    }

    useEffect(() => {
        if (quantityError || priceError) onExpand()
    }, [onExpand, quantityError, priceError])

    useLayoutEffect(() => {
        if (!expanded || !pendingPriceFocus.current) return
        pendingPriceFocus.current = false
        priceRef.current?.focus()
    }, [expanded])

    const expandToPrice = useCallback(() => {
        if (expanded) {
            priceRef.current?.focus()
            return
        }
        pendingPriceFocus.current = true
        onExpand()
    }, [expanded, onExpand])

    const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>, field: ItemField) => {
        const action = itemKeyAction({
            key: event.key,
            shiftKey: event.shiftKey,
            field,
            value: event.currentTarget.value,
            index,
            rowCount,
        })
        if (!action) return

        event.preventDefault()
        switch (action.type) {
            case 'addItem':
                return onAddItem()
            case 'expandToPrice':
                return expandToPrice()
            case 'focusOwn':
                return ownInputs[action.field].current?.focus()
            case 'removeRow':
                return onRemove()
            case 'focusSibling':
                return onFocusField(index + action.delta, field)
        }
    }

    const handleFoldClick = () => {
        if (expanded) {
            onToggle()
            return
        }
        pendingPriceFocus.current = true
        onExpand()
    }

    return { nameRef, quantityRef, priceRef, handleKeyDown, handleFoldClick }
}
