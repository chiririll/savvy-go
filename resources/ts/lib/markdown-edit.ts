// Pure editing helpers for MarkdownField: (text, selection) -> (text, selection).

export interface MarkdownEdit {
    text: string
    start: number
    end: number
}

export type ListKind = 'bullet' | 'task'

const BULLET = /^(\s*)[-*+] (?!\[[ xX]\] )/
const TASK = /^(\s*)[-*+] \[[ xX]\] /
const NUMBERED = /^(\s*)\d+\. /
const ANY_ITEM = /^(\s*)([-*+]|\d+\.) (\[[ xX]\] )?/

/** Toggles an inline marker (`**`, `*`, `` ` ``) around the selection. */
export function toggleWrap(edit: MarkdownEdit, marker: string): MarkdownEdit {
    const { text, start, end } = edit
    const n = marker.length
    const selected = text.slice(start, end)

    // Markers just outside the selection.
    const before = text.slice(start - n, start)
    const after = text.slice(end, end + n)
    const single = n === 1
    const outside =
        before === marker &&
        after === marker &&
        !(single && (text[start - 2] === marker || text[end + 1] === marker))
    if (outside) {
        return { text: text.slice(0, start - n) + selected + text.slice(end + n), start: start - n, end: end - n }
    }

    // Markers inside the selection.
    const inside =
        selected.length >= 2 * n &&
        selected.startsWith(marker) &&
        selected.endsWith(marker) &&
        !(single && (selected[1] === marker || selected[selected.length - 2] === marker))
    if (inside) {
        const inner = selected.slice(n, selected.length - n)
        return { text: text.slice(0, start) + inner + text.slice(end), start, end: start + inner.length }
    }

    return {
        text: text.slice(0, start) + marker + selected + marker + text.slice(end),
        start: start + n,
        end: end + n,
    }
}

export const toggleBold = (edit: MarkdownEdit) => toggleWrap(edit, '**')
export const toggleItalic = (edit: MarkdownEdit) => toggleWrap(edit, '*')
export const toggleCode = (edit: MarkdownEdit) => toggleWrap(edit, '`')

function stripPrefix(line: string): string {
    return line.replace(TASK, '$1').replace(BULLET, '$1').replace(NUMBERED, '$1')
}

/** Turns the lines touched by the selection into a bullet or task list, or back into plain lines. */
export function toggleList(edit: MarkdownEdit, kind: ListKind): MarkdownEdit {
    const { text, start, end } = edit
    const lineStart = text.lastIndexOf('\n', start - 1) + 1
    // A selection that ends right after a newline does not touch the next line.
    const probe = end > start && text[end - 1] === '\n' ? end - 1 : end
    const nl = text.indexOf('\n', probe)
    const lineEnd = nl === -1 ? text.length : nl

    const lines = text.slice(lineStart, lineEnd).split('\n')
    const already = kind === 'task' ? TASK : BULLET
    const targets = lines.filter((l) => l.trim() !== '')
    const remove = targets.length > 0 && targets.every((l) => already.test(l))
    const prefix = kind === 'task' ? '- [ ] ' : '- '

    const out = lines.map((line) => {
        if (remove) return stripPrefix(line)
        if (line.trim() === '' && lines.length > 1) return line
        const m = /^\s*/.exec(line)?.[0] ?? ''
        return m + prefix + stripPrefix(line).slice(m.length)
    })
    const block = out.join('\n')
    const next = text.slice(0, lineStart) + block + text.slice(lineEnd)

    if (start === end) {
        const delta = block.length - (lineEnd - lineStart)
        return { text: next, start: Math.max(lineStart, start + delta), end: Math.max(lineStart, start + delta) }
    }
    return { text: next, start: lineStart, end: lineStart + block.length }
}

const URL_LIKE = /^(https?:\/\/|mailto:)\S+$/i

/** Wraps the selection as a link and selects the part the user types next. */
export function insertLink(edit: MarkdownEdit): MarkdownEdit {
    const { text, start, end } = edit
    const selected = text.slice(start, end)
    const head = text.slice(0, start)
    const tail = text.slice(end)

    if (selected === '') {
        return { text: `${head}[](url)${tail}`, start: start + 1, end: start + 1 }
    }
    if (URL_LIKE.test(selected)) {
        return { text: `${head}[](${selected})${tail}`, start: start + 1, end: start + 1 }
    }
    const urlStart = start + selected.length + 3
    return { text: `${head}[${selected}](url)${tail}`, start: urlStart, end: urlStart + 3 }
}

/**
 * Enter inside a list item: starts the next item, or ends the list when the item is empty.
 * Returns null when the caret is not in a list item, so the browser handles Enter.
 */
export function continueList(edit: MarkdownEdit): MarkdownEdit | null {
    const { text, start, end } = edit
    if (start !== end) return null

    const lineStart = text.lastIndexOf('\n', start - 1) + 1
    const nl = text.indexOf('\n', start)
    const lineEnd = nl === -1 ? text.length : nl
    const line = text.slice(lineStart, lineEnd)

    const m = ANY_ITEM.exec(line)
    if (!m) return null
    const prefixLen = m[0].length
    if (start - lineStart < prefixLen) return null

    if (line.length === prefixLen) {
        // Empty item: remove the marker and leave the list.
        const next = text.slice(0, lineStart) + text.slice(lineEnd)
        return { text: next, start: lineStart, end: lineStart }
    }

    const [, indent, marker, task] = m
    const nextMarker = /^\d+\.$/.test(marker) ? `${parseInt(marker, 10) + 1}.` : marker
    const insert = `\n${indent}${nextMarker} ${task ? '[ ] ' : ''}`
    const caret = start + insert.length
    return { text: text.slice(0, start) + insert + text.slice(start), start: caret, end: caret }
}
