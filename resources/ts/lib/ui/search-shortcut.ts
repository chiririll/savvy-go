/** The slice of a KeyboardEvent that decides whether `/` opens the search palette. */
export interface SearchShortcutEvent {
    key: string
    ctrlKey: boolean
    metaKey: boolean
    altKey: boolean
    repeat: boolean
    isComposing: boolean
    target: EventTarget | null
}

function isEditable(target: EventTarget | null): boolean {
    const element = target as Partial<HTMLElement> | null
    if (!element?.tagName) return false
    return ['INPUT', 'TEXTAREA', 'SELECT'].includes(element.tagName) || element.isContentEditable === true
}

/**
 * `/` opens the palette, but not while typing, with a modal open, with a modifier held,
 * on key repeat or during IME composition.
 */
export function shouldOpenSearchPalette(event: SearchShortcutEvent, modalOpen: boolean): boolean {
    return (
        event.key === '/' &&
        !event.ctrlKey &&
        !event.metaKey &&
        !event.altKey &&
        !event.repeat &&
        !event.isComposing &&
        !modalOpen &&
        !isEditable(event.target)
    )
}
