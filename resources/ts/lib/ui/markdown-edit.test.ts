import { describe, expect, it } from 'vitest'

import {
    continueList,
    insertLink,
    toggleBold,
    toggleCode,
    toggleItalic,
    toggleList,
    type MarkdownEdit,
} from './markdown-edit'

// `|` marks the caret, `[` `]` mark a selection.
function parse(s: string): MarkdownEdit {
    if (s.includes('|')) {
        const i = s.indexOf('|')
        return { text: s.replace('|', ''), start: i, end: i }
    }
    const a = s.indexOf('[')
    const b = s.indexOf(']') - 1
    return { text: s.replace('[', '').replace(']', ''), start: a, end: b }
}

function show(e: MarkdownEdit | null): string {
    if (!e) return 'null'
    if (e.start === e.end) return e.text.slice(0, e.start) + '|' + e.text.slice(e.start)
    return e.text.slice(0, e.start) + '[' + e.text.slice(e.start, e.end) + ']' + e.text.slice(e.end)
}

describe('inline wrap', () => {
    it.each([
        [toggleBold, 'a [b] c', 'a **[b]** c'],
        [toggleBold, 'a | c', 'a **|** c'],
        [toggleBold, 'a **[b]** c', 'a [b] c'],
        [toggleBold, 'a [**b**] c', 'a [b] c'],
        [toggleItalic, 'a [b] c', 'a *[b]* c'],
        [toggleItalic, 'a *[b]* c', 'a [b] c'],
        [toggleItalic, 'a **[b]** c', 'a ***[b]*** c'],
        [toggleItalic, 'a [**b**] c', 'a *[**b**]* c'],
        [toggleCode, 'run [ls] now', 'run `[ls]` now'],
        [toggleCode, 'run `[ls]` now', 'run [ls] now'],
    ])('%#', (fn, input, want) => {
        expect(show(fn(parse(input)))).toBe(want)
    })
})

describe('toggleList', () => {
    it.each([
        ['bullet', 'milk|', '- milk|'],
        ['bullet', 'mi|lk', '- mi|lk'],
        ['bullet', '[milk\nbread]', '[- milk\n- bread]'],
        ['bullet', '[- milk\n- bread]', '[milk\nbread]'],
        ['bullet', '[- milk\nbread]', '[- milk\n- bread]'],
        ['bullet', '[milk\n\nbread]', '[- milk\n\n- bread]'],
        ['bullet', '|', '- |'],
        ['bullet', 'one\n[two]\nthree', 'one\n[- two]\nthree'],
        ['bullet', '[one\n]two', '[- one]\ntwo'],
        ['bullet', '1. mi|lk', '- mi|lk'],
        ['task', 'milk|', '- [ ] milk|'],
        ['task', '- milk|', '- [ ] milk|'],
        ['task', '- [x] milk|', 'milk|'],
        ['bullet', '- [ ] milk|', '- milk|'],
    ] as const)('%s %#', (kind, input, want) => {
        expect(show(toggleList(parse(input), kind))).toBe(want)
    })

    it('removes task markers from a multi-line selection', () => {
        const text = '- [ ] a\n- [x] b'
        expect(toggleList({ text, start: 0, end: text.length }, 'task')).toEqual({ text: 'a\nb', start: 0, end: 3 })
    })

    it('keeps indentation', () => {
        expect(show(toggleList(parse('  [milk]'), 'bullet'))).toBe('[  - milk]')
    })
})

describe('insertLink', () => {
    it('puts the caret in the label when nothing is selected', () => {
        expect(insertLink(parse('a |'))).toEqual({ text: 'a [](url)', start: 3, end: 3 })
    })

    it('wraps the selection and selects the url placeholder', () => {
        const got = insertLink(parse('a [text] b'))
        expect(got.text).toBe('a [text](url) b')
        expect(got.text.slice(got.start, got.end)).toBe('url')
    })

    it('uses a selected address as the target', () => {
        expect(insertLink(parse('[https://example.com]'))).toEqual({
            text: '[](https://example.com)',
            start: 1,
            end: 1,
        })
    })
})

describe('continueList', () => {
    it.each([
        ['- milk|', '- milk\n- |'],
        ['- [ ] milk|', '- [ ] milk\n- [ ] |'],
        ['- [x] milk|', '- [x] milk\n- [ ] |'],
        ['  - milk|', '  - milk\n  - |'],
        ['* milk|', '* milk\n* |'],
        ['1. milk|', '1. milk\n2. |'],
        ['9. milk|', '9. milk\n10. |'],
        ['- mi|lk', '- mi\n- |lk'],
        ['a\n- |', 'a\n|'],
        ['- [ ] |', '|'],
        ['- a\n- |\n- c', '- a\n|\n- c'],
        ['plain|', 'null'],
        ['- |milk', '- \n- |milk'],
        ['-| milk', 'null'],
        ['[- a]', 'null'],
    ])('%#', (input, want) => {
        expect(show(continueList(parse(input)))).toBe(want)
    })
})
