import { describe, expect, it } from 'vitest'

import { flattenNote, highlightParts, noteSnippet, searchTerms } from './search-highlight'

describe('searchTerms', () => {
    it('splits on spaces, lower-cases, drops repeats, longest first', () => {
        expect(searchTerms('  Milk  eggs MILK ')).toEqual(['milk', 'eggs'])
        expect(searchTerms('mi milk')).toEqual(['milk', 'mi'])
        expect(searchTerms('')).toEqual([])
    })
})

describe('highlightParts', () => {
    it.each([
        ['Oat milk run', ['milk'], [['Oat ', false], ['milk', true], [' run', false]]],
        ['MILK and Milk', ['milk'], [['MILK', true], [' and ', false], ['Milk', true]]],
        ['молоко', ['мол'], [['мол', true], ['око', false]]],
        ['nothing', ['milk'], [['nothing', false]]],
        ['a.b', ['.'], [['a', false], ['.', true], ['b', false]]],
        ['text', [], [['text', false]]],
        ['', ['x'], []],
    ] as const)('%s', (text, terms, expected) => {
        expect(highlightParts(text, [...terms]).map((p) => [p.text, p.match])).toEqual(expected)
    })
})

describe('flattenNote', () => {
    it('drops list markers and joins lines', () => {
        expect(flattenNote('- milk\n- [ ] eggs\n  * [x] bread\n\n1. jam')).toBe('milk · eggs · bread · jam')
    })
})

describe('noteSnippet', () => {
    it('returns the note around the first hit', () => {
        expect(noteSnippet(['- oat milk\n- eggs'], ['milk'])).toBe('oat milk · eggs')
    })

    it('cuts a long note with ellipses on both sides', () => {
        const note = `${'a'.repeat(60)} milk ${'b'.repeat(100)}`
        const snippet = noteSnippet([note], ['milk'])!
        expect(snippet.startsWith('…')).toBe(true)
        expect(snippet.endsWith('…')).toBe(true)
        expect(snippet).toContain('milk')
        expect(snippet.length).toBeLessThan(100)
    })

    it('skips empty notes and notes without a hit', () => {
        expect(noteSnippet([null, '', 'bread', 'Milk 2 l'], ['milk'])).toBe('Milk 2 l')
        expect(noteSnippet(['bread'], ['milk'])).toBeNull()
        expect(noteSnippet(['milk'], [])).toBeNull()
    })
})
