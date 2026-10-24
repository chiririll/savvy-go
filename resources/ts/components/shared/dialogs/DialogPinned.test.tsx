// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { useState, type ReactNode } from 'react'
import { afterEach, describe, expect, it } from 'vitest'

import { DialogPinned, PinnedSlotContext } from './DialogPinned'

/** A stand-in for ResponsiveDialog: a body, and the pinned slot below it. */
function Slot({ children }: { children: ReactNode }) {
    const [slot, setSlot] = useState<HTMLElement | null>(null)
    return (
        <PinnedSlotContext.Provider value={slot}>
            <div data-testid="body">{children}</div>
            <div data-testid="pinned-slot" ref={setSlot} />
        </PinnedSlotContext.Provider>
    )
}

/** Owns state, like a step does: the pinned button changes what the body shows. */
function Counter() {
    const [count, setCount] = useState(0)
    return (
        <>
            <p>count {count}</p>
            <DialogPinned>
                <button onClick={() => setCount((c) => c + 1)}>add</button>
            </DialogPinned>
        </>
    )
}

afterEach(() => cleanup())

describe('DialogPinned', () => {
    it('draws its children into the pinned slot, outside the body', () => {
        render(
            <Slot>
                <DialogPinned>
                    <button>add</button>
                </DialogPinned>
            </Slot>,
        )
        const button = screen.getByRole('button', { name: 'add' })
        expect(screen.getByTestId('pinned-slot').contains(button)).toBe(true)
        expect(screen.getByTestId('body').contains(button)).toBe(false)
    })

    it('leaves state and handlers with the component that renders it', () => {
        render(
            <Slot>
                <Counter />
            </Slot>,
        )
        fireEvent.click(screen.getByRole('button', { name: 'add' }))
        expect(screen.getByText('count 1')).toBeTruthy()
    })

    it('renders in place outside a dialog', () => {
        render(
            <div data-testid="here">
                <DialogPinned>
                    <button>add</button>
                </DialogPinned>
            </div>,
        )
        expect(screen.getByTestId('here').contains(screen.getByRole('button', { name: 'add' }))).toBe(true)
    })
})
