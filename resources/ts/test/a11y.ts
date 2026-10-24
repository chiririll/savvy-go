// Axe rules for stories. A story's own a11y config replaces the global list instead of adding to it, so every list
// here repeats the base rules.

// Colour contrast is checked everywhere. STORYBOOK_CONTRAST=off (set before vitest) skips it while chasing another failure.
const base = [{ id: 'color-contrast', enabled: import.meta.env.STORYBOOK_CONTRAST !== 'off' }]

export const baseA11y = { test: 'error', config: { rules: base } } as const

/** For a story with an open Radix menu, select or popover: Radix hides the page behind it with aria-hidden while the
 * trigger keeps focus, which axe reports as aria-hidden-focus. */
export const overlayA11y = {
    config: { rules: [...base, { id: 'aria-hidden-focus', enabled: false }] },
} as const

/** For an open emoji picker: emoji-picker-react points aria-controls at an element it has not rendered yet. */
export const emojiPickerA11y = {
    config: { rules: [...base, { id: 'aria-valid-attr-value', enabled: false }] },
} as const

/** For a page whose tabs show only the active panel: Radix leaves aria-controls of the other triggers pointing at
 * panels it has not mounted, which axe reports as aria-valid-attr-value. */
export const unmountedTabsA11y = {
    config: { rules: [...base, { id: 'aria-valid-attr-value', enabled: false }] },
} as const

/** The tab pages above that also open a menu or select. */
export const unmountedTabsOverlayA11y = {
    config: { rules: [...base, { id: 'aria-valid-attr-value', enabled: false }, { id: 'aria-hidden-focus', enabled: false }] },
} as const
