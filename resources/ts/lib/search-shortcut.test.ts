// @vitest-environment jsdom
import { describe, expect, it } from 'vitest'

import { shouldOpenSearchPalette, type SearchShortcutEvent } from './search-shortcut'

const press = (overrides: Partial<SearchShortcutEvent> = {}): SearchShortcutEvent => ({
    key: '/',
    ctrlKey: false,
    metaKey: false,
    altKey: false,
    repeat: false,
    isComposing: false,
    target: document.body,
    ...overrides,
})

describe('shouldOpenSearchPalette', () => {
    it('opens on a bare slash', () => {
        expect(shouldOpenSearchPalette(press(), false)).toBe(true)
    })

    it('ignores other keys', () => {
        expect(shouldOpenSearchPalette(press({ key: 'k' }), false)).toBe(false)
    })

    it.each(['ctrlKey', 'metaKey', 'altKey', 'repeat', 'isComposing'] as const)('ignores %s', (flag) => {
        expect(shouldOpenSearchPalette(press({ [flag]: true }), false)).toBe(false)
    })

    it('ignores a slash typed into a field', () => {
        for (const tag of ['input', 'textarea', 'select']) {
            expect(shouldOpenSearchPalette(press({ target: document.createElement(tag) }), false)).toBe(false)
        }
        const editable = document.createElement('div')
        Object.defineProperty(editable, 'isContentEditable', { value: true })
        expect(shouldOpenSearchPalette(press({ target: editable }), false)).toBe(false)
    })

    it('still opens from a focused button', () => {
        expect(shouldOpenSearchPalette(press({ target: document.createElement('button') }), false)).toBe(true)
    })

    it('ignores a slash while a modal is open', () => {
        expect(shouldOpenSearchPalette(press(), true)).toBe(false)
    })
})
