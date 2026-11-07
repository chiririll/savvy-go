/**
 * Marking of a free-text search in text: the terms the backend matches (`internal/db/filter`),
 * cut out of the text so the palette can paint them, and the part of a line's note around the
 * first hit.
 */

export interface HighlightPart {
    text: string
    match: boolean
}

/** The words of `q`, longest first so a long term wins over a shorter one inside it. */
export function searchTerms(q: string): string[] {
    const words = q
        .split(/\s+/)
        .map((word) => word.trim())
        .filter(Boolean)
    return [...new Set(words.map((word) => word.toLowerCase()))].sort((a, b) => b.length - a.length)
}

const escapeRegExp = (value: string) => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

function termsPattern(terms: string[], flags: string): RegExp | null {
    return terms.length > 0 ? new RegExp(terms.map(escapeRegExp).join('|'), flags) : null
}

/** Splits `text` into parts, the ones matching any term (any case) flagged. */
export function highlightParts(text: string, terms: string[]): HighlightPart[] {
    const pattern = termsPattern(terms, 'giu')
    if (!pattern || !text) {
        return text ? [{ text, match: false }] : []
    }
    const parts: HighlightPart[] = []
    let last = 0
    for (const found of text.matchAll(pattern)) {
        if (found[0] === '') {
            continue
        }
        const start = found.index ?? 0
        if (start > last) {
            parts.push({ text: text.slice(last, start), match: false })
        }
        parts.push({ text: found[0], match: true })
        last = start + found[0].length
    }
    if (last < text.length) {
        parts.push({ text: text.slice(last), match: false })
    }
    return parts
}

/** A note as one line of plain text: list and task markers dropped, line breaks as " · ". */
export function flattenNote(note: string): string {
    return note
        .split(/\r?\n/)
        .map((line) => line.replace(/^\s*(?:[-*+]|\d+[.)])\s+(?:\[[ xX]\]\s+)?/, '').trim())
        .filter(Boolean)
        .join(' · ')
}

const BEFORE = 24
const AFTER = 56

/**
 * The part of the first note that holds a term, around the first hit, with "…" where it is cut.
 * Null when no note matches.
 */
export function noteSnippet(notes: Array<string | null | undefined>, terms: string[]): string | null {
    const pattern = termsPattern(terms, 'iu')
    if (!pattern) {
        return null
    }
    for (const note of notes) {
        if (!note) {
            continue
        }
        const flat = flattenNote(note)
        const found = pattern.exec(flat)
        if (!found) {
            continue
        }
        const start = Math.max(0, found.index - BEFORE)
        const end = Math.min(flat.length, found.index + found[0].length + AFTER)
        return `${start > 0 ? '…' : ''}${flat.slice(start, end).trim()}${end < flat.length ? '…' : ''}`
    }
    return null
}
