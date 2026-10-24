import { createContext, useContext, type ReactNode } from 'react'
import { createPortal } from 'react-dom'

/** The element above the footer of a ResponsiveDialog; null outside one, or before it is mounted. */
export const PinnedSlotContext = createContext<HTMLElement | null>(null)

/**
 * Content that stays above the footer buttons however far the body is scrolled, such as an
 * add form under a long list. It is drawn into the dialog's pinned slot, but belongs to the
 * component that renders it: state and handlers stay there. Outside a dialog it renders in place.
 */
export function DialogPinned({ children }: { children: ReactNode }) {
    const slot = useContext(PinnedSlotContext)
    return slot ? createPortal(children, slot) : <>{children}</>
}
